<script lang="ts">
import Button from '$components/buttons/Button.svelte';
import FilterInput from '$components/form/FilterInput.svelte';
import Input from '$components/form/Input.svelte';
import Modal from '$components/layers/Modal.svelte';
import SearchSelect from '$components/form/SearchSelect.svelte';
import { goto } from '$app/navigation';
import { Core, closeModal, openModal, tr } from '$core/store.svelte';
import Page from '$domain/Page.svelte';
import { ConfirmWarn, formatTime, Loading, Notify } from '$libs/helpers';
import { UsuariosService } from '../../security/users/users.svelte';
import { LAST_SYSTEM_PAGE_ID, WebpagesService, showcaseImageSrc, type IWebpage } from '$services/webpage/pages.svelte';
import WebpageConfig from './WebpageConfig.svelte';
    import T from '$components/misc/T.svelte';

  // Header top tabs (rendered by the Page shell). Switch on Core.pageOptionSelected.
  const pageOptions = [{ id: 1, name: 'Pages|Páginas' }, { id: 2, name: 'Config' }];
  const PAGE_FORM_MODAL_ID = 2;

  const pages = new WebpagesService(true);
  const usuarios = new UsuariosService();

  let filterText = $state('');
  let form = $state({} as IWebpage);

  // Status options shown in the form. Removal is handled by the Modal delete button.
  const statusOptions = [
    { i: 1, v: 'Active|Activo' },
    { i: 2, v: 'Published|Publicado' },
  ];

  // A system page (ID <= 14) is injected by the server and cannot be edited.
  const isSystemPage = (page: IWebpage) => page.ID > 0 && page.ID <= LAST_SYSTEM_PAGE_ID;

  const filteredPages = $derived.by(() => {
    const normalized = filterText.toLowerCase().trim();
    if (!normalized) return pages.records;
    return pages.records.filter(
      (page) =>
        page.Name.toLowerCase().includes(normalized) ||
        page.Route.toLowerCase().includes(normalized),
    );
  });

  const updatedByName = (page: IWebpage) => {
    if (!page.UpdatedBy) return '';
    const user = usuarios.usuariosMap.get(page.UpdatedBy);
    return user ? `${user.FirstName} ${user.LastName}`.trim() || user.Usuario : `User-${page.UpdatedBy}`;
  };

  const statusLabel = (status: number) =>
    status === 2 ? tr('Published|Publicado') : tr('Active|Activo');

  const newPage = () => {
    form = { ss: 1 } as IWebpage;
    openModal(PAGE_FORM_MODAL_ID);
  };

  const selectPage = (page: IWebpage) => {
    // System pages are read-only — clicking them does nothing.
    if (isSystemPage(page)) return;
    form = { ...page };
    openModal(PAGE_FORM_MODAL_ID);
  };

  // The pencil opens the builder for this page's content (distinct from the card
  // body, which edits the page metadata). Works for system pages too — only their
  // name/route metadata is read-only, not their content.
  const editPageContent = (page: IWebpage) => goto(`/webpage-builder/${page.ID}`);

  const savePage = async (isDelete?: boolean) => {
    if ((form.Name || '').trim().length < 3) {
      Notify.failure(tr('Name must be at least 3 characters.|El nombre debe tener al menos 3 caracteres.'));
      return;
    }
    if (!isDelete && (!form.Route || !form.Route.startsWith('/') || form.Route === '/')) {
      Notify.failure(tr('Route must start with "/" and cannot be the root.|La ruta debe iniciar con "/" y no puede ser la raíz.'));
      return;
    }
    if (isDelete) form.ss = 0;

    Loading.standard(tr('Saving page...|Guardando página...'));
    try {
      await pages.postAndSync([form]);
    } catch (error) {
      Notify.failure(error as string);
      Loading.remove();
      return;
    }
    Loading.remove();
    form = {} as IWebpage;
    closeModal(PAGE_FORM_MODAL_ID);
  };
</script>

<Page title="Pages|Páginas" options={pageOptions}>
  {#if Core.pageOptionSelected === 1}
    <!-- Toolbar: filter + new (the view tabs live in the header, not here) -->
    <div class="flex items-center mb-10 gap-12">
      <FilterInput label="Filter pages|Filtrar páginas"
        css="w-full md:w-260" icon="icon-[fa--search]" bind:value={filterText}
      />
      <Button name="New|Nuevo" label="Shows the form to create a new page."
        color="green" icon="icon-[fa--plus]" hideNameOnMobile css="ml-auto"
        onClick={newPage}
      />
    </div>

    <!-- Pages as cards (same layout as product categories) -->
    <div class="w-full cards-scroll-container" aria-label="Website pages list">
      <div class="w-full flex flex-wrap gap-12 content-start">
        {#each filteredPages as page (page.ID)}
          <div class="_1 w-260"
            class:_system={isSystemPage(page)}
            role="button" tabindex="0"
            onkeydown={(ev) => {
              if (ev.key === 'Enter' || ev.key === ' ') { ev.preventDefault(); selectPage(page); }
            }}
            onclick={() => selectPage(page)}
          >
            <!-- Thumbnail: the page's showcase screenshot (saved from the builder),
                 or a placeholder icon when the page has none yet. -->
            <div class="_thumb flex items-center justify-center">
              {#if showcaseImageSrc(page)}
                <img class="w-full h-full object-cover" src={showcaseImageSrc(page)} alt={page.Name} loading="lazy" />
              {:else}
                <i class="icon-[fa--image] fs28 c-steel op-50"></i>
              {/if}
            </div>

            <!-- Pencil: opens the builder for this page's content (shown on hover). -->
            <button class="_pencil" type="button"
              aria-label={tr('Edit page content|Editar contenido de la página')}
              onclick={(ev) => { ev.stopPropagation(); editPageContent(page); }}
            >
              <i class="icon-[fa--pencil]"></i>
            </button>

            <div class="px-12 py-10">
              <div class="flex items-center justify-between mb-4">
                <div class="fs17 ff-semibold"><T text={page.Name}/></div>
                <span class="fs12 px-6 py-1 rounded-full {page.ss === 2 ? 'bg-green-100 c-green' : 'bg-blue-100 c-blue'}">
                  {statusLabel(page.ss)}
                </span>
              </div>
              <div class="fs14 c-blue mb-6">{page.Route}</div>
              <div class="flex">
	              <div class="fs13 c-steel">{updatedByName(page) || '—'}</div>
								<div class="text-sm c-steel ml-auto">{page.upd ? formatTime(page.upd, 'M-d h:m') : '—'}</div>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </div>

    <Modal title="PAGE|PÁGINA" id={PAGE_FORM_MODAL_ID} size={5}
      onClose={() => { form = {} as IWebpage; closeModal(PAGE_FORM_MODAL_ID); }}
      onSave={() => savePage()}
      onDelete={form.ID > 0 ? () => {
        ConfirmWarn(
          tr('Delete Page|Eliminar Página'),
          tr(`Delete "${form.Name}"?|¿Eliminar "${form.Name}"?`),
          'SI', 'NO', () => savePage(true),
        );
      } : undefined}
    >
      <div class="grid grid-cols-12 gap-10 p-6" aria-label="Page form: name, route, status">
        <Input label="Name|Nombre" css="col-span-12" saveOn={form} save="Name" required={true} />
        <Input label="Route|Ruta" css="col-span-12" saveOn={form} save="Route" />
        <SearchSelect label="Status|Estado" css="col-span-12 mb-6"
          saveOn={form} save="ss" keyId="i" keyName="v" options={statusOptions}
        />
      </div>
    </Modal>
  {/if}

  {#if Core.pageOptionSelected === 2}
    <WebpageConfig />
  {/if}
</Page>

<style>
  .cards-scroll-container {
    height: calc(100dvh - 160px);
    max-height: calc(100dvh - 160px);
    overflow-y: auto;
    overflow-x: hidden;
    padding: 2px;
  }

  ._1 {
    position: relative;
    overflow: hidden;
    background-color: rgb(255, 255, 255);
    border-radius: 8px;
    box-shadow: rgba(0, 0, 0, 0.1) 0px 4px 6px -1px, rgba(0, 0, 0, 0.06) 0px 2px 4px -1px;
    transition: box-shadow 0.15s ease, transform 0.15s ease;
  }
  ._1:hover {
    transform: translateY(-2px);
    box-shadow: rgba(0, 0, 0, 0.18) 0px 10px 20px -4px, rgba(0, 0, 0, 0.1) 0px 4px 8px -2px;
  }

  /* Thumbnail placeholder until image upload lands. */
  ._thumb {
    height: 160px;
    overflow: hidden;
    background: linear-gradient(135deg, #eef2f7, #e2e8f0);
    border-bottom: 1px solid rgba(0, 0, 0, 0.06);
  }
  /* Crop the showcase screenshot to the thumbnail box, anchored to the top so the
     page's header/hero is what shows. */
  ._thumb img {
    object-position: top;
  }

  /* Pencil button: hidden until the card is hovered/focused. */
  ._pencil {
    position: absolute;
    top: 8px;
    right: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.95);
    box-shadow: rgba(0, 0, 0, 0.15) 0px 2px 5px;
    color: #334155;
    opacity: 0;
    transition: opacity 0.15s ease, background 0.15s ease;
  }
  ._1:hover ._pencil,
  ._1:focus-within ._pencil { opacity: 1; }
  ._pencil:hover { background: #fff; color: #2563eb; }

  /* System pages: only their metadata is read-only (the card body), so dim the
     body affordance slightly — the pencil still works to edit page content. */
  ._system { cursor: default; }
</style>
