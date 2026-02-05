<script lang="ts">
  import ImageEffect from '../../components/ImageEffect.svelte';

  const baseUrl = '/store-images/';
  
  const examples = [
    {
      title: 'Curved Reflection (Standard Layout)',
      src: 'modern-bookstore-interior_3-2_white-wood_light.avif',
      layout: 'curve-right',
      color: '#ffffff',
      children: true,
      text: 'Vast Knowledge',
      subtext: 'Our libraries host over 50,000 titles from various disciplines and genres.',
      align: 'left'
    },
    {
      title: 'Curved Layout + Duotone Matrix Effect',
      src: 'modern-bookstore-interior_3-2_white-wood_light.avif',
      layout: 'curve-convex-right',
      effect: 'duotone-matrix',
      color: '#4c2d82',
      color2: '#ffa500',
      children: true,
      text: 'Combined Styles',
      subtext: 'Using curve-convex layout with a duotone-matrix visual treatment.',
      align: 'left'
    },
    {
      title: 'Slash Layout (Image on Right)',
      src: 'industrial-mercantile-clothing-store_3-2_white-wood_light.avif',
      layout: 'slash-right',
      color: '#f3f4f6',
      children: true,
      text: 'Our Mission',
      subtext: 'We design and build high-quality architectural projects with a focus on sustainability and modern design.',
      align: 'left'
    },
    {
      title: 'Fade Layout (Soft Space)',
      src: 'shopping-online-payment_3-2_brown-black_light.avif',
      layout: 'fade-right',
      color: '#ffffff',
      intensity: 1,
      children: true,
      text: 'Secure Payments',
      subtext: 'Experience the best online shopping security with our encrypted payment gateway.',
      align: 'left'
    },
    {
      title: 'Fade Layout + Glass Effect',
      src: 'loccitane-skincare-products_4-5_green-white_light.avif',
      layout: 'fade-left',
      effect: 'glass',
      color: '#ffffff',
      intensity: 0.8,
      blur: 5,
      children: true,
      text: 'Premium Skincare',
      subtext: 'Natural ingredients for your daily routine.',
      align: 'right'
    },
    {
      title: 'Vignette Effect (Full Image)',
      src: 'minimalist-workspace-laptop-phone_1-1_white-black_light.avif',
      effect: 'vignette',
      color: '#000000',
      intensity: 0.7,
      aspectRatio: '1/1'
    },
    {
      title: 'Duotone Effect (Standard)',
      src: 'woman-yellow-armchair-smartphone_4-3_yellow-red_light.avif',
      effect: 'duotone',
      color: '#4c55d5',
      color2: '#f5f6ff',
      aspectRatio: '16/9'
    }
  ];
</script>

<div class="p-8 max-w-7xl mx-auto bg-white min-h-screen">
  <h1 class="text-4xl font-bold mb-4">ImageEffect Component Showcase</h1>
  <p class="text-gray-600 mb-12">Demonstrating the power of separating <b>Layout</b> (composition) from <b>Effect</b> (visual treatment).</p>

  <div class="space-y-24">
    {#each examples as ex}
      <section>
        <h2 class="text-2xl font-semibold mb-6 text-gray-800">{ex.title}</h2>
        <div class="rounded-2xl overflow-hidden shadow-2xl border border-gray-100">
          <ImageEffect 
            src={baseUrl + ex.src} 
            layout={ex.layout}
            effect={ex.effect} 
            color={ex.color} 
            color2={ex.color2}
            intensity={ex.intensity}
            blur={ex.blur}
            aspectRatio={ex.aspectRatio || '21/9'}
            css={ex.children ? "min-h-[400px] flex items-center p-12" : ""}
          >
            {#if ex.children}
              <div class={["max-w-md w-full", 
                ex.align === 'right' ? 'ml-auto text-right' : 
                ex.align === 'center' ? 'mx-auto text-center' : 'mr-auto text-left'
              ].join(" ")}>
                <h3 class="text-4xl font-black mb-4 tracking-tight" style="color: {ex.effect === 'overlay' || ex.effect === 'vignette' || (ex.layout === 'fade-right' && ex.color === '#000000') ? 'white' : '#1a1a1a'}">
                  {ex.text}
                </h3>
                <p class="text-lg opacity-90" style="color: {ex.effect === 'overlay' || ex.effect === 'vignette' || (ex.layout === 'fade-right' && ex.color === '#000000') ? 'white' : '#4b5563'}">
                  {ex.subtext}
                </p>
                <button class="mt-8 px-8 py-3 bg-black text-white font-bold rounded-lg hover:bg-gray-800 transition-colors">
                  Learn More
                </button>
              </div>
            {/if}
          </ImageEffect>
        </div>
        <div class="mt-4 bg-gray-50 p-4 rounded-lg font-mono text-sm overflow-x-auto">
          <code>
            &lt;ImageEffect src="{ex.src}" {ex.layout ? `layout="${ex.layout}"` : ''} {ex.effect ? `effect="${ex.effect}"` : ''} /&gt;
          </code>
        </div>
      </section>
    {/each}
  </div>
</div>

<style>
  :global(body) {
    background-color: #f3f4f6;
  }
</style>
