<script lang="ts">
  type KeyValuePrimitive = string | number;
  type KeyValueFormatter = (value: KeyValuePrimitive) => string;

  interface KeyValueStripProps {
    css?: string;
    labelCss?: string;
    textCss?: string;
    lineCss?: string;
    label1?: string;
    value1?: KeyValuePrimitive;
    getContent1?: KeyValueFormatter;
    label2?: string;
    value2?: KeyValuePrimitive;
    getContent2?: KeyValueFormatter;
    label3?: string;
    value3?: KeyValuePrimitive;
    getContent3?: KeyValueFormatter;
    label4?: string;
    value4?: KeyValuePrimitive;
    getContent4?: KeyValueFormatter;
    label5?: string;
    value5?: KeyValuePrimitive;
    getContent5?: KeyValueFormatter;
    label6?: string;
    value6?: KeyValuePrimitive;
    getContent6?: KeyValueFormatter;
    label7?: string;
    value7?: KeyValuePrimitive;
    getContent7?: KeyValueFormatter;
    label8?: string;
    value8?: KeyValuePrimitive;
    getContent8?: KeyValueFormatter;
    label9?: string;
    value9?: KeyValuePrimitive;
    getContent9?: KeyValueFormatter;
    label10?: string;
    value10?: KeyValuePrimitive;
    getContent10?: KeyValueFormatter;
  }

  const formatValue = (value: KeyValuePrimitive): string => String(value);

  let {
    css = '',
    labelCss = '',
    textCss = '',
    lineCss = '',
    label1,
    value1,
    getContent1 = formatValue,
    label2,
    value2,
    getContent2 = formatValue,
    label3,
    value3,
    getContent3 = formatValue,
    label4,
    value4,
    getContent4 = formatValue,
    label5,
    value5,
    getContent5 = formatValue,
    label6,
    value6,
    getContent6 = formatValue,
    label7,
    value7,
    getContent7 = formatValue,
    label8,
    value8,
    getContent8 = formatValue,
    label9,
    value9,
    getContent9 = formatValue,
    label10,
    value10,
    getContent10 = formatValue,
  }: KeyValueStripProps = $props();

  // Normalize the repeated prop API into a single list so rendering stays minimal.
  const keyValueRows = $derived(
    [
      { label: label1, value: value1, getContent: getContent1 },
      { label: label2, value: value2, getContent: getContent2 },
      { label: label3, value: value3, getContent: getContent3 },
      { label: label4, value: value4, getContent: getContent4 },
      { label: label5, value: value5, getContent: getContent5 },
      { label: label6, value: value6, getContent: getContent6 },
      { label: label7, value: value7, getContent: getContent7 },
      { label: label8, value: value8, getContent: getContent8 },
      { label: label9, value: value9, getContent: getContent9 },
      { label: label10, value: value10, getContent: getContent10 },
    ].filter(({ label, value }) => label && !!value)
  );
</script>

<div class={"flex flex-wrap items-center gap-x-16 gap-y-8 " + css}>
  {#each keyValueRows as keyValueRow}
    <div class="flex flex-col items-center min-w-0">
      <div class={"strip-label " + (labelCss || 'text-xs color-label leading-[1.1] w-full ff-semibold')}>
        {keyValueRow.label}
      </div>
      <div class={lineCss || 'h-1 w-24 bg-slate-200 rounded-full'}></div>
      <div class={textCss || 'text-sm'}>
        {keyValueRow.getContent(keyValueRow.value as KeyValuePrimitive)}
      </div>
    </div>
  {/each}
</div>

<style>
	.strip-label {
		border-bottom: 1px solid #5f488f45;
		margin-bottom: -1px;
    z-index: 1;
	}
</style>
