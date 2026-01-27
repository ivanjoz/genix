<script lang="ts">
  import { onMount } from 'svelte';
  import { Env } from '$core/env';

  interface Props {
    amount: number;
    email: string;
    order?: string;
    onSuccess?: (id: string, type: 'token' | 'order') => void;
    onError?: (error: any) => void;
  }

  let { amount, email, order, onSuccess, onError }: Props = $props();

  let culqiInitialized = false;

  async function loadCulqi() {
    if ((window as any).CulqiCheckout) return (window as any).CulqiCheckout;
    return new Promise((resolve) => {
      const script = document.createElement('script');
      script.src = 'https://js.culqi.com/checkout-js';
      script.async = true;
      script.onload = () => {
        resolve((window as any).CulqiCheckout);
      };
      document.body.appendChild(script);
    });
  }

        async function initCulqi() {
          if (culqiInitialized) return;
          if (amount <= 0) {
            console.warn("Amount must be greater than 0 to initialize Culqi");
            return;
          }
          
          const CulqiCheckout = await loadCulqi();
          
          const settings: any = {
            title: Env.empresa.Nombre || 'Genix Store',
            currency: 'PEN',
            amount: Math.round(amount),
          };
    
          if (order) {
            settings.order = order;
          }
      
          const client = {
            email: email || '',
          };
      
          const paymentMethods = {
            tarjeta: true,
            yape: !!order,
            billetera: !!order,
            bancaMovil: !!order,
            agente: !!order,
            cuotealo: !!order,
          };
      
          const options = {
            lang: 'auto',
            modal: false,
            container: '#culqi-container',
            paymentMethods: paymentMethods,
            paymentMethodsSort: Object.keys(paymentMethods),
          };
      
          const appearance = {
            theme: "default",
            menuType: "sidebar",
            hiddenCulqiLogo: true,
            hiddenBannerContent: true,
            hiddenBanner: false,
            hiddenToolBarAmount: false,
            buttonCardPayText: 'Pagar S/ ' + (amount / 100).toFixed(2),
          };
    
          const config = { settings, client, options, appearance };
          const publicKey = Env.empresa.CulqiLlave || 'pk_test_...';
    const Culqi = new (CulqiCheckout as any)(publicKey, config);

    (window as any).culqi = () => {
      if (Culqi.token) {
        onSuccess?.(Culqi.token.id, 'token');
        Culqi.close();
      } else if (Culqi.order) {
        onSuccess?.(Culqi.order, 'order');
        Culqi.close();
      } else {
        console.error('Culqi Error:', Culqi.error);
        if (onError) onError(Culqi.error);
        else alert("Error en el pago: " + (Culqi.error?.user_message || "Desconocido"));
      }
    };

    Culqi.open();
    culqiInitialized = true;
  }

  onMount(() => {
    setTimeout(initCulqi, 100);
  });
</script>

<div id="culqi-container" class="grow-1 w-full min-h-[500px]">
  <div class="flex flex-col items-center justify-center h-full text-gray-400">
    <i class="icon-spin5 animate-spin text-[32px] mb-2"></i>
    <p>Cargando pasarela de pago...</p>
  </div>
</div>
