<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import Checkbox from '$components/form/Checkbox.svelte';
import T from '$components/misc/T.svelte';
import { companyFlagsCatalog, toggleCompanyFlag } from './company-flags';
import { saveCompanyParameters, type EmpresaParametrosService } from './empresas.svelte';

  const { service }: { service: EmpresaParametrosService } = $props()

  // Controlled checkboxes: the value is membership in a list, not a property of an object, which
  // is what Checkbox's saveOn binding would need.
  const isFlagChecked = (flagID: number) => (service.empresa.Flags || []).includes(flagID)

  const setFlag = (flagID: number, isChecked: boolean) => {
    service.empresa.Flags = toggleCompanyFlag(service.empresa.Flags, flagID, isChecked)
  }
</script>

<section class="rounded-[12px] border border-slate-200 bg-white p-16 shadow-sm"
  aria-label="Company flags table: each row turns one business rule on for this company">
  <div class="flex items-center gap-10 mb-4">
    <div class="h3 ff-bold flex-1 min-w-0"><T text="Company Flags|Flags de la Empresa" /></div>
    <!-- The same record and the same endpoint as the parameters form beside it, so either Save
         writes both panels. It is repeated here because this is where the flags are edited. -->
    <Button color="blue" icon="icon-[fa--floppy-o]" name="Save|Guardar"
      label="Saves the checked flags on the company."
      onClick={() => saveCompanyParameters(service.empresa)} />
  </div>
  <div class="text-slate-500 mb-14">
    <T text="A checked flag turns its rule on for the whole company.|Un flag marcado activa su regla para toda la empresa." />
  </div>

  <table class="w-full border-collapse">
    <tbody>
      {#each companyFlagsCatalog as section (section.label)}
        <tr>
          <th class="text-left ff-bold text-slate-600 bg-slate-100 px-10 py-6
            border-y border-slate-200"><T text={section.label} /></th>
        </tr>
        {#each section.flags as flag (flag.id)}
          <tr class="border-b border-slate-100">
            <!-- One control per row rather than a box in one cell and inert text in the next: the
                 Checkbox already lays its box and its label out as two columns, and keeping them
                 in one control means the name is a click target and announces once. -->
            <td class="px-10 py-6">
              <Checkbox label={flag.name} underlineOnHover={true}
                checked={isFlagChecked(flag.id)}
                onToggle={(isChecked) => setFlag(flag.id, isChecked)} />
            </td>
          </tr>
        {/each}
      {/each}
    </tbody>
  </table>
</section>
