# Créditos extra para GET (`company_extra_credits_24h`)

**Estado: implementado.** Este documento queda como el registro de diseño. Tres cosas salieron
distintas de lo planeado y están anotadas donde corresponde:

1. `makeCompanyCreditBudgetResponse` recibe el techo del pool como parámetro en vez de leer
   `core.Env`: `Env` es un puntero nil en los tests unitarios y la función revienta si lo toca. El
   test que lo destapó ya existía.
2. Las cifras del pool viajan al panel **aunque no haya presupuesto del mes en curso**, saltándose el
   retorno anticipado de `makeCompanyCreditBudgetResponse`. Esa es precisamente la company que vive
   del pool, así que esconderlo dejaría el modo de sólo lectura invisible justo cuando está en uso.
3. Los fixtures de los tests de Rust se escribieron primero con `daily = 2` y fallaron: la puerta
   diaria de usuario es `daily / 2`, así que la primera request ya salía del pool. Es el caso de la
   company de un solo usuario, el mismo que justifica que el pool no tenga sub-tope por usuario, y
   ahora los fixtures lo dicen explícitamente (`EXTRA_DAILY = 4`).

Nomenclatura: **extra** en todas partes — `company_extra_credits_24h`, `extra_daily`,
`day_extra_used`, `EXTRA_CREDIT_FLAG`, `ExtraRemainingCPU`. La versión anterior de este documento
decía «reserva» y era un nombre inventado por mí que no aparecía en ninguna parte del código ni de la
configuración; una palabra por concepto.

## Lo que se pide

```toml
# Credits used when exausted normal credit quota only for GET requests
company_extra_credits_24h = 50000
```

Cuando una company agota su cuota, le quedan 50 000 créditos diarios que **sólo** puede gastar en
GET. Modo degradado de sólo lectura en vez de un 429 en toda la aplicación.

## Estado verificado del código

1. **`company_extra_credits_24h` no lo lee nadie.** La clave sólo aparece en `config.toml`.
2. **Las cuatro claves `_24h` que ya estaban tampoco.** `load_scope_limits`
   (`server_utils/src/config.rs:420`) lee únicamente `_10s` y `_1h`. El techo diario real sale de la
   tabla `company_credit_budget`. Detalle importante: `config.toml` está en `.gitignore`, y
   `config.example.toml` —el que sí está en el repo— **no tiene ninguna clave `_24h`**. Así que no
   hay nada deprecado que borrar del repositorio: son cuatro líneas muertas en tu archivo local.
3. **`CreditLimits` sólo tiene `ten_seconds` y `hour`** (`quota.rs:39`). Diario y mensual se evalúan
   contra `StoredBudget.daily` y `StoredBudget.monthly_ceiling`, que vienen de la tabla.
4. **Las siete puertas de `admit_at`**: company 10s → user 10s → company 1h → user 1h → company
   diario (`budget.stored.daily`) → user diario (`daily / 2`) → mensual (`monthly_violation`).
5. **Una company sin fila de presupuesto está negada del todo.** `StoredBudget::default()` deja
   `daily = 0` y `monthly_ceiling = 0`, y `exceeds(0, 2, 0)` es `true`. Nada siembra esa fila al
   crear la company: sólo la escribe el panel SaaS (`backend/config/company_credit_budget.go`). Es
   el estado «sin créditos» más frecuente que existe hoy, y el que esta feature vuelve utilizable.
6. **El frame de cargo no lleva el método.** El daemon no puede distinguir un GET de un POST, y por
   diseño no conoce la tabla de rutas.
7. **Un GET se cobra en dos frames**: la base antes del handler (`enforceAccessAndCredits`) y una
   liquidación después si la respuesta pasó de 8 KB (`chargeGetResponseTopUp`). Los dos llevan la
   marca.
8. **No hay helpers muertos en la ruta de créditos.** Revisé `APICPUBaseCredits`,
   `ChargeAPIAccessOnly`, `ChargeAPICredits`, `InferenceCredits`, `ChargeInferenceUsage`,
   `IsCreditRateLimitError` y `MakeCreditRateLimitResponse`: todos tienen llamador real. Lo único
   que este cambio deja obsoleto son las cuatro claves del punto 2.

---

## Transporte: bit 15 de `route_id`

El frame no cambia de tamaño. `MAX_ROUTE_ID` es 16 383 (14 bits) y ambos lados ya validan que los
dos bits altos vengan en cero, así que hay hueco pagado.

```rust
const EXTRA_CREDIT_FLAG: u16 = 0x8000;

let encoded_route = u16::from_be_bytes([payload[6], payload[7]]);
let extra_credits_allowed = encoded_route & EXTRA_CREDIT_FLAG != 0;
let route_id = encoded_route & !EXTRA_CREDIT_FLAG;
// La validación que ya existe sigue viva y ahora hace doble trabajo: el bit 14 no está
// asignado, así que un frame que lo traiga puesto cae aquí como InvalidRouteID.
if route_id > MAX_ROUTE_ID { return Err(ProtocolError::InvalidRouteID(route_id)); }
```

Se despeja **sólo** el bit 15, no con una máscara de 14 bits, precisamente para que el chequeo de
`MAX_ROUTE_ID` siga siendo el guardián del bit reservado. Cero variantes de error nuevas, cero
`CHARGE_PAYLOAD_SIZE`, cero churn en los tests de layout.

En Go el flag se aplica después de convertir a `uint16`, así que el `int16` del parámetro no se ve:

```go
const extraCreditFlag = uint16(0x8000)

encodedRoute := uint16(routeID)          // routeID sigue validado en 0..maxChargeRouteID
if extraCreditsAllowed { encodedRoute |= extraCreditFlag }
binary.BigEndian.PutUint16(payload[6:8], encodedRoute)
```

Degradación si sólo sube una mitad: daemon nuevo + backend viejo → el bit llega apagado → los extra
nunca aplican, que es el comportamiento actual. Daemon viejo + backend nuevo → `InvalidRouteID` →
frame rechazado → el backend falla cerrado → 503. Ruidoso e inmediato, y de todas formas ya tienen
que subir juntos por el dominio HMAC `:v6`.

El riesgo real de sobrecargar el campo es que `credits_blob.rs` vuelve a desplazar el route id
(`route_id << 2`) al serializar. Ahí llega ya limpio —el flag se despeja en `parse_charge`, antes de
que el `Request` exista— pero hay que fijarlo con un test, porque es el único sitio donde un flag
filtrado corrompería silenciosamente la contabilidad por ruta.

---

## Semántica

**Las puertas de ráfaga (10s y 1h) no se relajan nunca.** Existen para proteger la máquina y una
avalancha de GET es exactamente lo que protegen. Un GET pagado con extra compite por el bucket de 10
segundos igual que cualquier otro, y **suma a `hour_used` y a los buckets**. Si no lo hiciera, una
company en modo extra tendría ráfaga infinita.

**El consumo extra no toca `day_used` ni `month_used`.** Se contabiliza aparte, en su propio
contador. Dos consecuencias buenas: `daily - day_used` sigue significando lo que significa hoy (así
que la puerta del POST no cambia ni una línea), y el techo mensual comprado no se mueve nunca.

**Los extra no tienen sub-tope por usuario.** El pool es de la company y un solo usuario puede
agotarlo. Es deliberado: es una asignación gratuita de company, y si la puerta diaria de usuario
(`daily / 2`) también bloqueara el pool, una company de un solo usuario —la mayoría hoy— no llegaría
nunca a tocarlo. Las ráfagas siguen acotando el ritmo.

### El flujo en `admit_at`

1. Cuatro puertas de ráfaga. **Niegan siempre**, marcado o no.
2. Puertas diaria de company, diaria de usuario y mensual, como hoy. Si pasan las tres → cargo
   normal, sin cambios.
3. Si alguna de las tres niega, y `extra_credits_allowed`, y `requested.inference == 0`, y
   `day_extra_used + requested.cpu <= extra_daily` → se admite con extra:
   - `day_extra_used += cpu`, `month_extra_used += cpu`
   - buckets y `hour_used` sí se cargan
   - `day_used` y `month_used` **no**
   - las filas de uso (`increment_usage`) y el agregado de plataforma sí se escriben: son el registro
     de lo que el sistema realmente sirvió y de ahí lee el panel por ruta.
4. Si no cabe, la negación original se devuelve tal cual. El cliente ve el mismo 429 de siempre.

El orden importa: los extra se consultan **después** de que la cuota normal haya negado, nunca antes.
Un GET que cabe en la cuota normal no gasta extra.

`SubjectState::charge` hoy carga buckets, `hour_used` y `day_used` juntos; necesita un parámetro
(`count_daily: bool`) para el caso 3. Dos llamadores.

Sólo CPU, un `u64`, no `Credits`: el número de configuración es uno solo y la inferencia se cobra por
`ChargeInferenceUsage`, que nunca va marcado. Un frame marcado que pida inferencia no recibe
relajación en ninguna dimensión (condición explícita en el paso 3).

### Las dos columnas: una es la cuota, la otra es contabilidad

**La que aplica la política es diaria.** `extra_credits_24h` es un tope por día, así que lo que el
daemon consulta en cada request es `day_extra_used` contra el día local de negocio. No hay ningún
tope mensual de extra.

**`month_extra_used` no es una segunda cuota, es un término de corrección.** `ensure_budget`
reconstruye `month_used` **sumando las filas de uso del mes** (`load_range` desde el inicio del mes)
al arrancar en frío, y esas filas incluyen el consumo extra, porque los cargos extra sí se escriben
en el registro de uso. Sin restarlo, cada reinicio del daemon le comería a la company parte de su
entitlement pagada. Con la columna, la recuperación es exacta:

```rust
month_used = sum(monthly_rows).saturating_sub(month_extra_used)
```

Las dos se guardan por los periodos que la fila **ya tiene** (`usage_day_period` y
`usage_month_start_day`), así que no hace falta ninguna columna de periodo nueva y la regla de
lectura no cambia: periodo distinto al actual → contador cero.

#### Alternativa: la cifra extra en la fila diaria de uso

`increment_usage` escribe cuatro filas por cargo — {usuario, agregado de company} × {frame de 5
minutos, frame diario} — así que **ya existe una fila diaria de company por día**. Si la cifra extra
viviera ahí, el total del mes saldría del mismo `load_range` que `ensure_budget` ya hace, y el extra
de hoy sería el de la última fila: **una columna en vez de dos, y historial por día gratis** para
`CompanyCreditCalendar.svelte`.

No es la recomendación por dónde cae el cambio, no por el diseño:

- `load_range` decodifica con un solo `rows::<(i32, Vec<u8>)>()` para las dos tablas, así que la
  forma de fila es compartida: `credit_usage_user` tendría que ganar la columna aunque no la use.
- `UsageRecord`, `UsageSnapshot` y `upsert` entran en la máquina de versiones y snapshots absolutos
  —`mark_flushed`, `is_clean`, `merge_loaded`—, que es el código con los invariantes más delicados
  del módulo, y hoy esta feature no lo toca en absoluto.

Las dos columnas en `company_credit_budget` se quedan enteras dentro de la ruta de presupuesto que
este cambio ya modifica. Es una preferencia por contención del radio de impacto; si el historial
por día vale más que eso, la alternativa es mejor y no cambia ninguna otra decisión del plan.

---

## Fases

### Fase 1 — configuración
- `config.rs`: `rate_limit.company_extra_credits_24h` → `LimitPolicy.extra_daily: u64`, env
  `RATE_LIMIT_COMPANY_EXTRA_CREDITS_24H`.
- Default **0 = apagado**. Es la única omisión segura: un pool de extra adivinado es crédito
  regalado. Con 0, el paso 3 no existe y el comportamiento es exactamente el de hoy.
- `validate_policy` no le impone orden contra `hour`: son ejes distintos.
- Añadirla a `config.example.toml` con el comentario de que 0 la desactiva. Borrar del `config.toml`
  local las cuatro claves `_24h` muertas y dejar una línea diciendo que el techo diario vive en
  `company_credit_budget`.

### Fase 2 — transporte
- `protocol.rs`: `EXTRA_CREDIT_FLAG`, `Request.extra_credits_allowed`, despeje del bit antes del
  chequeo de `MAX_ROUTE_ID`.
- `credits.go`: `extraCreditFlag`, parámetro nuevo en `encodeCharge` / `Charge` /
  `chargeConfiguredCredits`.
- `main-handlers.go`: la marca sale de `chargedMethodFor(args.Method, funcPath) == "GET"` y de nada
  más. `chargeGetResponseTopUp` la lleva por definición. `ChargeInferenceUsage` nunca.

### Fase 3 — decisión
- `CompanyBudgetState`: `day_extra_used`, `month_extra_used`, reset en el rollover de día y de mes
  que ya existe.
- `SubjectState::charge(requested, count_daily)`.
- `admit_at`: el paso 3 de arriba.
- `StoredBudgetUsage` + `usage_snapshot` publican las dos columnas; `select_budget` las lee para la
  recuperación en frío; `ensure_budget` resta `month_extra_used` de la suma de filas.
- `company_credit_budget`: `day_extra_cpu_used int64`, `month_extra_cpu_used int64` en
  `backend/core/types/company_credit_budget.go` (struct y tabla) y en los INSERT de uso de
  `storage.rs`.

### Fase 4 — visibilidad
- `backend/config/company_credit_usage.go`: `ExtraCPU` y `ExtraRemainingCPU`, al lado de
  `DailyRemainingCPU`.
- `CompanyCreditMeters.svelte`: tramo extra en el medidor.
- Un log en el backend cuando una request se sirvió con extra. Es la señal para soporte, y por ahora
  la única: el frame de respuesta no cambia de forma y el cliente no se entera.

### Fuera de alcance, anotado
Hoy, si la base de un GET se admite y la respuesta pasa de 8 KB, `chargeGetResponseTopUp` puede ser
**rechazada** y el 429 reemplaza un body ya generado. Los extra hacen ese caso más probable, porque
la base se admitirá justo en el borde. Arreglarlo bien es que la liquidación cobre y nunca rechace
—el trabajo ya se hizo, contarlo es mejor que negarlo y perder la métrica— pero es un cambio de
comportamiento independiente. No lo toco aquí.

---

## Tests

- **Rust, decisión**: un GET marcado pasa cuando la puerta diaria niega; el mismo frame sin marcar es
  rechazado; con la mensual negando, también pasa; el pool agotado vuelve a negar; los extra **no**
  saltan el bucket de 10s ni el de 1h; un frame marcado con `inference > 0` no se relaja;
  `extra_daily = 0` desactiva todo; un GET que cabe en la cuota normal no toca `day_extra_used`.
- **Rust, contabilidad**: tras un cargo extra, `day_used` y `month_used` no se movieron y
  `hour_used` sí; `day_extra_used` se reinicia al cambiar el día **local de negocio**
  (`time_frame::local_unix_day`, no el UTC); `ensure_budget` recupera `month_used` exacto restando
  `month_extra_used`.
- **Rust, layout**: el bit 15 se despeja y no llega a `credits_blob::encode` (el test que impide que
  un flag filtrado corrompa la contabilidad por ruta); un frame con el bit 14 puesto →
  `InvalidRouteID`; un route_id legítimo nunca activa el flag.
- **Go**: `encodeCharge` marca sólo cuando el método cobrado es GET;
  `TestPostAndPutAreIndistinguishable` se extiende a «ninguno de los dos lleva la marca»; los
  vectores de bytes del frame.

## Riesgos

1. **La marca la decide el backend y el daemon la obedece.** Si un POST la llevara por error,
   escribiría gratis. Mitigación: se deriva de `chargedMethodFor(...) == "GET"` y de nada más, con
   test que lo fija; y el daemon no relaja nada si `inference > 0`.
2. **Regalo acotado sólo por el número.** 50 000 créditos con `GET` base = 2 son ~25 000 lecturas
   gratis al día por encima de lo pagado, indefinidamente. La palanca es el número, no el diseño: con
   5 000 el modo degradado sigue siendo usable y el regalo queda acotado. **Decidir el valor antes de
   desplegar.**
3. **Despliegue acoplado.** Backend y daemon suben juntos, como ya ocurre con `:v6`. Sigue pendiente
   de la sesión anterior: el daemon del puerto 14013 es pre-`v6`.
4. **Esto da lectura, no escritura.** Una company sin fila de presupuesto podrá leer pero seguirá sin
   poder escribir nada. Si lo que se quiere es que una company nueva funcione, eso es sembrar la fila
   al crearla — otro cambio.
