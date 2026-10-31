<script lang="ts" generics="T extends object">
import SearchSelect from '#components/form/SearchSelect.svelte'
import { pcgeAccountLabel, pcgeAccountsForKind } from '#core/pcge.ts'

// The PCGE account picker of every form that classifies a movement: a supply, a purchase line,
// an expense, an asset. It offers only the accounts the kind may use (PCGE_KIND_* in #core/pcge.ts),
// the same rule the backend validates with, so the user cannot pick one the save would refuse.
interface Props {
  saveOn: T
  save: keyof T
  kind: number
  label?: string
  css?: string
  required?: boolean
  disabled?: boolean
  placeholder?: string
}

const {
  saveOn, save, kind, label = "Accounting account|Cuenta contable", css = "", required = false,
  disabled = false, placeholder,
}: Props = $props()

const accountOptions = $derived(pcgeAccountsForKind(kind)
  .map(account => ({ code: account.code, label: pcgeAccountLabel(account.code) })))
</script>

<SearchSelect saveOn={saveOn} save={save} css={css} label={label} keyId="code" keyName="label"
  options={accountOptions} required={required} disabled={disabled} placeholder={placeholder} />
