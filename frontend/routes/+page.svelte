<script lang="ts">
  import T from '#components/misc/T.svelte';
  // Welcome/Home page
</script>

<div class="flex flex-col items-center justify-center min-h-[80vh] text-center">
  <div class="max-w-2xl mx-auto p-8">
    <!-- Logo -->
    <div class="mb-8">
      <div
        class="w-24 h-24 mx-auto bg-linear-to-br from-indigo-600 to-indigo-800
				rounded-2xl flex items-center justify-center shadow-xl"
      >
        <span class="text-5xl">🏢</span>
      </div>
    </div>

    <!-- Title -->
    <h1 class="text-5xl font-bold text-fg mb-4">
      <T text="Welcome to Genix|Bienvenido a Genix" />
    </h1>

    <!-- Subtitle -->
    <p class="text-xl text-fg-soft mb-8">
      <T text="Comprehensive Business Management System|Sistema de Gestión Empresarial Integral" />
    </p>

    <!-- Description -->
    <div class="bg-surface rounded-lg shadow-lg p-6 mb-8">
      <p class="text-fg-soft leading-relaxed">
        <T text="This is the new frontend version built with|Esta es la nueva versión del frontend desarrollada con" />
        <strong class="text-accent-fg">Svelte 5</strong>
        <T text="and|y" />
        <strong class="text-accent-fg">Tailwind CSS</strong>.
        <T text="The side menu has been migrated from Solid.js with a modern, responsive design.|El menú lateral ha sido migrado desde Solid.js con un diseño moderno y responsive." />
      </p>
    </div>

    <!-- Features -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
      <div class="bg-surface rounded-lg shadow-md p-6">
        <div class="text-3xl mb-3">⚡</div>
        <h3 class="font-semibold text-fg mb-2"><T text="Fast|Rápido" /></h3>
        <p class="text-sm text-fg-soft">
          <T text="Built with Svelte 5 for maximum performance|Construido con Svelte 5 para máximo rendimiento" />
        </p>
      </div>

      <div class="bg-surface rounded-lg shadow-md p-6">
        <div class="text-3xl mb-3">📱</div>
        <h3 class="font-semibold text-fg mb-2">
          Responsive
        </h3>
        <p class="text-sm text-fg-soft">
          <T text="Works perfectly on desktop, tablet, and mobile|Funciona perfectamente en desktop, tablet y móvil" />
        </p>
      </div>

      <div class="bg-surface rounded-lg shadow-md p-6">
        <div class="text-3xl mb-3">🎨</div>
        <h3 class="font-semibold text-fg mb-2">
          <T text="Modern|Moderno" />
        </h3>
        <p class="text-sm text-fg-soft">
          <T text="Updated design with Tailwind CSS|Diseño actualizado con Tailwind CSS" />
        </p>
      </div>
    </div>

    <!-- Quick Links -->
    <div class="flex flex-wrap justify-center gap-4">
      <a
        href="/system/empresas"
        class="inline-flex items-center px-6 py-3 bg-accent-solid hover:bg-accent-solid
				text-white font-semibold rounded-lg shadow-md transition-colors"
      >
        <span class="mr-2">🏢</span>
        <T text="View Companies|Ver Empresas" />
      </a>

      <a
        href="/production/products"
        class="inline-flex items-center px-6 py-3 bg-surface
				hover:bg-surface-soft text-fg
				font-semibold rounded-lg shadow-md border border-line
				transition-colors"
      >
        <span class="mr-2">📦</span>
        <T text="Products|Productos" />
      </a>
    </div>

    <!-- Footer Info -->
    <div class="mt-12 text-sm text-fg-muted">
      <p><T text="Use the side menu to navigate the different sections|Usa el menú lateral para navegar por las diferentes secciones" /></p>
      <p class="mt-2">
        <T text="Press|Presiona" />
        <kbd class="px-2 py-1 bg-surface-strong rounded text-xs font-mono">⚙️</kbd>
        <T text="in the header to change the theme|en el header para cambiar el tema" />
      </p>
    </div>
  </div>
</div>
