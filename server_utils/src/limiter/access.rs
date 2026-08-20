//! One user's authorization grants, held in memory and answered without touching ScyllaDB.
//!
//! This is the half of the charge frame that is not about credits. The Go router used to answer it
//! itself with a `SELECT accesos_computed FROM users` behind a five-minute in-process cache, which
//! is nearly free on a VPS and expensive on Lambda: every new execution environment starts cold and
//! pays a database round trip on the authorization path before the handler runs. This daemon is the
//! one process that is always resident, so the answer lives here and the frame that was already
//! going out carries the question.
//!
//! What this module deliberately does not know: access *names*, which route maps to which access,
//! that `access_list.yml` exists at all. It answers "does this user hold any of these grants" and
//! every policy rule around that — an unmapped GET being free, `POST.user-self` needing no access,
//! user 1 bypassing the check entirely — stays in Go, where the catalogue is embedded.

use crate::limiter::storage::StoredUserAccess;

/// How many required grants one frame can carry. `access_list.yml` maps at most two accesses to any
/// one backend route today, so this is 2x headroom for eight bytes of frame. A route that needs a
/// fifth is refused Go-side at encode time, never here — a rejected frame is indistinguishable from
/// the daemon being down and would surface as a 503 instead of as the bug it is.
pub const MAX_REQUIRED_ACCESS: usize = 4;

/// Payload of opcode `0x06`: `[company:u24][user:u24]`.
pub const INVALIDATE_ACCESS_PAYLOAD_SIZE: usize = 6;

/// Which cached grants to drop. `user_id == 0` is the wildcard, since user ids start at 1.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub struct AccessInvalidation {
    pub company_id: i32,
    /// Zero means every cached user of the company.
    pub user_id: i32,
}

/// Decodes one invalidation. Only the company is required to be real: a wildcard is the point of
/// user zero, and a stale entry for a user the backend has since deleted is still worth dropping.
pub fn parse_access_invalidation(
    payload: &[u8; INVALIDATE_ACCESS_PAYLOAD_SIZE],
) -> anyhow::Result<AccessInvalidation> {
    let company_id = read_u24(&payload[0..3]) as i32;
    let user_id = read_u24(&payload[3..6]) as i32;
    if company_id <= 0 {
        anyhow::bail!("company_id must be positive");
    }
    Ok(AccessInvalidation {
        company_id,
        user_id,
    })
}

fn read_u24(bytes: &[u8]) -> u32 {
    (u32::from(bytes[0]) << 16) | (u32::from(bytes[1]) << 8) | u32::from(bytes[2])
}

/// `users.status` value that means the user exists and may act. Anything else — 0 from a soft
/// delete, or a value some future migration invents — is refused.
const ACTIVE_USER_STATUS: i8 = 1;

/// Why a request was refused. Distinct from a credit violation: these say something about the
/// session or the user, not about how much the tenant has spent.
#[derive(Clone, Copy, Debug, PartialEq, Eq)]
pub enum AccessDenial {
    /// The user holds none of the required accesses at the required level.
    NoAccess = 2,
    /// No such user in this company. The token decoded and its HMAC checked, so this means the
    /// user was hard-deleted, or the token names a company the user does not belong to.
    UnknownUser = 3,
    /// The row exists but `status != 1`.
    InactiveUser = 4,
}

impl AccessDenial {
    /// Rides in the reply frame's `detail` field. 0 and 1 are reserved for "not requested" and
    /// "granted", which is why these start at 2.
    pub fn detail_code(self) -> u16 {
        self as u16
    }
}

/// One user's cached authorization state.
///
/// Two bytes per grant and nothing else: a user holding every access in today's catalogue costs 68
/// bytes of payload, less than the `HashMap` entry wrapped around it.
///
/// A flat bitmap keyed by acceso id would be denser only in theory. The packed u16 admits ids up to
/// 16383, so two bits each is 4 KB per user — sixty times worse than the sorted list for any
/// realistic grant count — and sizing it to today's largest id (34) would bake in a ceiling that the
/// next `access_list.yml` entry silently overflows. `parse_charge` already refuses to know the route
/// table for exactly this reason.
#[derive(Clone, Debug, PartialEq, Eq)]
pub struct UserAccessState {
    /// Sorted packed grants, `acceso_id << 2 | (nivel - 1)`. Empty when the user was not found.
    grants: Box<[u16]>,
    status: i8,
    /// Whether the row exists at all. A miss is cached like a hit, or a token naming a deleted user
    /// would re-query ScyllaDB on every request it sends.
    found: bool,
    /// Unix seconds of the load. Reloaded once older than the configured TTL.
    loaded_at: i64,
}

impl UserAccessState {
    /// Builds the cached state from the row, or from its absence.
    pub fn from_row(row: Option<StoredUserAccess>, loaded_at: i64) -> anyhow::Result<Self> {
        let Some(row) = row else {
            return Ok(Self {
                grants: Box::from([]),
                status: 0,
                found: false,
                loaded_at,
            });
        };
        Ok(Self {
            grants: decode_grants(&row.grants_blob)?,
            status: row.status,
            found: true,
            loaded_at,
        })
    }

    pub fn is_fresh(&self, unix_seconds: i64, ttl_seconds: i64) -> bool {
        // A clock that went backwards reads as stale rather than as fresh forever.
        unix_seconds >= self.loaded_at && unix_seconds - self.loaded_at <= ttl_seconds
    }

    /// The verdict for one frame's required grants. `None` is granted.
    ///
    /// Identity is checked before grants: a request from a user who no longer exists is not a
    /// permission problem and must not be reported as one, because the two produce different HTTP
    /// answers on the Go side (401 versus 403).
    pub fn verdict(&self, required: &[u16; MAX_REQUIRED_ACCESS]) -> Option<AccessDenial> {
        if !self.found {
            return Some(AccessDenial::UnknownUser);
        }
        if self.status != ACTIVE_USER_STATUS {
            return Some(AccessDenial::InactiveUser);
        }
        // Slots fill from index 0 and zero terminates, so `take_while` walks exactly the ones the
        // caller filled. Any one of them being held is enough — a route mapped to several accesses
        // is satisfied by any of them, matching what the Go gate did.
        let holds_any = required
            .iter()
            .take_while(|grant| **grant != 0)
            .any(|grant| self.holds(*grant));
        if holds_any {
            None
        } else {
            Some(AccessDenial::NoAccess)
        }
    }

    /// Whether any granted level at or above `required` exists inside the same acceso id bucket.
    ///
    /// `required | 0b11` is that bucket's ceiling — `acceso_id << 2 | 3`, nivel 4 — because the low
    /// two bits are the level and nothing else. This is `hasPackedAccesoInRange` from
    /// `backend/core/responses.go`, ported deliberately unchanged: the same algorithm over the same
    /// representation is what keeps the two processes from drifting into disagreeing about what a
    /// grant means.
    fn holds(&self, required: u16) -> bool {
        let bucket_ceiling = required | 0b11;
        match self.grants.binary_search(&required) {
            Ok(_) => true,
            Err(index) => self
                .grants
                .get(index)
                .is_some_and(|grant| *grant <= bucket_ceiling),
        }
    }
}

/// Decodes the `accesos_computed` blob into sorted packed grants.
///
/// **The blob is little-endian.** Every integer in this daemon's wire protocol is big-endian and
/// this column is not: `backend/genix-orm/scylla/converter.go` writes each element with
/// `binary.LittleEndian.PutUint16`. Getting this backwards would not fail — it would silently
/// authorize the wrong things.
///
/// Sorted here even though both Go writers already sort. It is at most a few dozen elements, and it
/// means a blob some future write path leaves out of order degrades into a wrong answer for one
/// user rather than into a binary search over unsorted data.
fn decode_grants(blob: &[u8]) -> anyhow::Result<Box<[u16]>> {
    if blob.len() % 2 != 0 {
        anyhow::bail!(
            "accesos_computed blob length {} is not a whole number of u16 grants",
            blob.len()
        );
    }
    let mut grants: Vec<u16> = blob
        .chunks_exact(2)
        .map(|pair| u16::from_le_bytes([pair[0], pair[1]]))
        .collect();
    grants.sort_unstable();
    grants.dedup();
    Ok(grants.into_boxed_slice())
}

#[cfg(test)]
mod tests {
    use super::*;

    /// Mirrors `core.makeAccesoNivelUint16`.
    fn packed(acceso_id: u16, nivel: u16) -> u16 {
        (acceso_id << 2) | (nivel - 1)
    }

    fn blob(grants: &[u16]) -> Vec<u8> {
        grants.iter().flat_map(|g| g.to_le_bytes()).collect()
    }

    fn required(grants: &[u16]) -> [u16; MAX_REQUIRED_ACCESS] {
        let mut slots = [0_u16; MAX_REQUIRED_ACCESS];
        slots[..grants.len()].copy_from_slice(grants);
        slots
    }

    fn active(grants: &[u16]) -> UserAccessState {
        UserAccessState::from_row(
            Some(StoredUserAccess {
                grants_blob: blob(grants),
                status: 1,
            }),
            1_000,
        )
        .unwrap()
    }

    /// The endianness is the one thing in this module that could be wrong without failing, so it is
    /// asserted against bytes written by hand rather than by the encoder under test.
    #[test]
    fn the_blob_is_little_endian() {
        // packed(34, 4) = 34<<2 | 3 = 139 = 0x008B.
        let state = UserAccessState::from_row(
            Some(StoredUserAccess {
                grants_blob: vec![0x8B, 0x00],
                status: 1,
            }),
            0,
        )
        .unwrap();
        assert_eq!(state.grants.as_ref(), &[139]);
        assert!(state.holds(packed(34, 1)));
        // Big-endian would have read 0x8B00 = 35584 = acceso 8896, which no catalogue contains.
        assert!(!state.holds(packed(8896, 1)));
    }

    #[test]
    fn an_odd_length_blob_is_corrupt() {
        assert!(decode_grants(&[0x01, 0x00, 0x02]).is_err());
        assert!(decode_grants(&[]).is_ok());
    }

    #[test]
    fn grants_are_sorted_and_deduplicated_whatever_the_blob_holds() {
        let state = active(&[packed(9, 1), packed(2, 4), packed(9, 1), packed(1, 2)]);
        assert_eq!(
            state.grants.as_ref(),
            &[packed(1, 2), packed(2, 4), packed(9, 1)]
        );
    }

    /// The level rule: a grant satisfies every level at or below it, and never leaks into the
    /// neighbouring acceso id.
    #[test]
    fn a_higher_level_satisfies_a_lower_requirement() {
        let state = active(&[packed(7, 4)]);
        for nivel in 1..=4 {
            assert!(state.verdict(&required(&[packed(7, nivel)])).is_none());
        }

        let view_only = active(&[packed(7, 1)]);
        assert!(view_only.verdict(&required(&[packed(7, 1)])).is_none());
        for nivel in 2..=4 {
            assert_eq!(
                view_only.verdict(&required(&[packed(7, nivel)])),
                Some(AccessDenial::NoAccess)
            );
        }
    }

    /// The bucket ceiling must not spill: holding acceso 8 at any level says nothing about 7 or 9.
    #[test]
    fn a_grant_never_satisfies_a_neighbouring_acceso() {
        let state = active(&[packed(8, 4)]);
        assert_eq!(
            state.verdict(&required(&[packed(7, 1)])),
            Some(AccessDenial::NoAccess)
        );
        assert_eq!(
            state.verdict(&required(&[packed(9, 1)])),
            Some(AccessDenial::NoAccess)
        );
        assert!(state.verdict(&required(&[packed(8, 1)])).is_none());
    }

    /// A route mapped to several accesses is satisfied by any one of them.
    #[test]
    fn any_filled_slot_is_enough_and_zero_terminates() {
        let state = active(&[packed(20, 2)]);
        assert!(
            state
                .verdict(&required(&[packed(3, 2), packed(20, 2)]))
                .is_none()
        );
        assert_eq!(
            state.verdict(&required(&[packed(3, 2), packed(4, 2)])),
            Some(AccessDenial::NoAccess)
        );
        // A grant sitting past a zero slot is not read: the gate fills from index 0.
        let mut sparse = [0_u16; MAX_REQUIRED_ACCESS];
        sparse[1] = packed(20, 2);
        assert_eq!(state.verdict(&sparse), Some(AccessDenial::NoAccess));
    }

    /// An empty required list never reaches here — the caller skips the check — but if it did, no
    /// grant can satisfy it, and failing closed is the only safe reading.
    #[test]
    fn no_required_grants_is_refused_rather_than_waved_through() {
        assert_eq!(
            active(&[packed(1, 4)]).verdict(&[0; MAX_REQUIRED_ACCESS]),
            Some(AccessDenial::NoAccess)
        );
    }

    /// Identity outranks permission: these become 401s on the Go side, not 403s.
    #[test]
    fn identity_is_judged_before_grants() {
        let missing = UserAccessState::from_row(None, 0).unwrap();
        assert_eq!(
            missing.verdict(&required(&[packed(1, 1)])),
            Some(AccessDenial::UnknownUser)
        );

        let soft_deleted = UserAccessState::from_row(
            Some(StoredUserAccess {
                grants_blob: blob(&[packed(1, 4)]),
                status: 0,
            }),
            0,
        )
        .unwrap();
        assert_eq!(
            soft_deleted.verdict(&required(&[packed(1, 1)])),
            Some(AccessDenial::InactiveUser)
        );
    }

    #[test]
    fn a_user_with_no_grants_at_all_is_a_permission_denial() {
        // Distinct from UnknownUser: the row exists and is active, it just grants nothing.
        assert_eq!(
            active(&[]).verdict(&required(&[packed(1, 1)])),
            Some(AccessDenial::NoAccess)
        );
    }

    #[test]
    fn freshness_expires_and_survives_a_backward_clock() {
        let state = active(&[]);
        assert!(state.is_fresh(1_000, 600));
        assert!(state.is_fresh(1_600, 600));
        assert!(!state.is_fresh(1_601, 600));
        // Earlier than the load: something reset the clock, so treat the entry as unusable.
        assert!(!state.is_fresh(999, 600));
    }

    #[test]
    fn an_invalidation_decodes_its_two_ids_and_its_wildcard() {
        let invalidation =
            parse_access_invalidation(&[0x00, 0x00, 0x07, 0x00, 0x01, 0x2C]).unwrap();
        assert_eq!(invalidation.company_id, 7);
        assert_eq!(invalidation.user_id, 300);
        // User zero is the wildcard, not an error: user ids start at 1.
        assert_eq!(
            parse_access_invalidation(&[0x00, 0x00, 0x07, 0x00, 0x00, 0x00])
                .unwrap()
                .user_id,
            0
        );
        assert!(parse_access_invalidation(&[0; INVALIDATE_ACCESS_PAYLOAD_SIZE]).is_err());
    }

    #[test]
    fn detail_codes_match_the_documented_contract() {
        assert_eq!(AccessDenial::NoAccess.detail_code(), 2);
        assert_eq!(AccessDenial::UnknownUser.detail_code(), 3);
        assert_eq!(AccessDenial::InactiveUser.detail_code(), 4);
    }
}
