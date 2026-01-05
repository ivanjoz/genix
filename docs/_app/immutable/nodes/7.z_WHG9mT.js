import { $ as proxy, L as delegate, M as append, P as from_html, Q as sibling, Tt as __toESM, X as child, Z as first_child, _t as reset, at as state, lt as pop, rt as set, st as user_derived, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { i as closeModal, t as Core } from "../chunks/BwrZ3UQO.js";
import { a as throttle, o as require_notiflix_aio_3_2_8_min, t as Notify } from "../chunks/DwKmqXYU.js";
import { a as formatTime } from "../chunks/DOFkf9MZ.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
import { n as UsuariosService, r as postUsuario, t as PerfilesService } from "../chunks/BKsRj9OC.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as Modal } from "../chunks/iQLQpuQM.js";
import "../chunks/BOmyLNB5.js";
import { t as SearchCard } from "../chunks/BHNEJas0.js";
var import_notiflix_aio_3_2_8_min = /* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min());
var root_2 = from_html(`<div class="grid grid-cols-24 gap-10 p-6"><!> <!> <!> <!> <!> <!> <!> <!> <!></div>`);
var root_1 = from_html(`<div class="h-full"><div class="flex items-center justify-between mb-6"><div class="i-search mr-16 w-256"><div><i class="icon-search"></i></div> <input class="w-full" autocomplete="off" type="text"/></div> <div class="flex items-center"><button class="bx-green" aria-label="Agregar usuario"><i class="icon-plus"></i></button></div></div> <!></div> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	const usuariosService = new UsuariosService();
	const perfilesService = new PerfilesService();
	let filterText = state("");
	let usuarioForm = state(proxy({}));
	async function saveUsuario(isDelete) {
		const form = get(usuarioForm);
		if ((form.usuario?.length || 0) < 4 || (form.nombres?.length || 0) < 4) {
			Notify.failure("El usuario y el nombre deben tener al menos 4 caracteres.");
			return;
		}
		if (form.password1) form.password1 = form.password1.trim();
		if (form.password2) form.password2 = form.password2.trim();
		if (!form.id || form.password1) {
			let err = "";
			if ((form.password1?.length || 0) < 6) err = "El password tiene menos de 6 caracteres.";
			else if (form.password1 !== form.password2) err = "Los password no coinciden.";
			if (err) {
				Notify.failure(err);
				return;
			}
		}
		import_notiflix_aio_3_2_8_min.Loading.standard("Creando/Actualizando Usuario...");
		try {
			const result = await postUsuario(form);
			if (isDelete) usuariosService.removeUsuario(form.id);
			else {
				if (!form.id) form.id = result.id;
				usuariosService.updateUsuario(form);
			}
			closeModal(1);
		} catch (error) {
			Notify.failure(error);
		}
		import_notiflix_aio_3_2_8_min.Loading.remove();
	}
	const columns = [
		{
			header: "ID",
			headerCss: "w-54",
			cellCss: "text-center ff-bold",
			getValue: (e) => e.id
		},
		{
			header: "Usuario",
			highlight: true,
			cellCss: "px-6 c-purple",
			getValue: (e) => e.usuario
		},
		{
			header: "Nombres",
			highlight: true,
			getValue: (e) => `${e.nombres} ${e.apellidos || ""}`
		},
		{
			header: "Email",
			cellCss: "px-6",
			getValue: (e) => e.email
		},
		{
			header: "Estado",
			headerCss: "w-80",
			cellCss: "text-center",
			getValue: (e) => e.ss
		},
		{
			header: "Actualizado",
			headerCss: "w-144",
			cellCss: "px-6 nowrap",
			getValue: (e) => formatTime(e.upd, "Y-m-d h:n")
		},
		{
			header: "...",
			headerCss: "w-42",
			cellCss: "t-c",
			id: "actions",
			buttonEditHandler: (rec) => {
				set(usuarioForm, { ...rec }, true);
				Core.openModal(1);
			}
		}
	];
	Page($$anchor, {
		title: "Usuarios",
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = first_child(fragment_1);
			var div_1 = child(div);
			var div_2 = child(div_1);
			var input = sibling(child(div_2), 2);
			input.__keyup = (ev) => {
				ev.stopPropagation();
				throttle(() => {
					set(filterText, (ev.target.value || "").toLowerCase().trim(), true);
				}, 150);
			};
			reset(div_2);
			var div_3 = sibling(div_2, 2);
			var button = child(div_3);
			button.__click = (ev) => {
				ev.stopPropagation();
				set(usuarioForm, { ss: 1 }, true);
				Core.openModal(1);
			};
			reset(div_3);
			reset(div_1);
			VTable(sibling(div_1, 2), {
				get columns() {
					return columns;
				},
				get data() {
					return usuariosService.usuarios;
				},
				css: "w-full",
				maxHeight: "calc(80vh - 13rem)",
				get filterText() {
					return get(filterText);
				},
				getFilterContent: (e) => [
					e.usuario,
					e.nombres,
					e.apellidos,
					e.email
				].filter((x) => x).join(" ").toLowerCase()
			});
			reset(div);
			var node_1 = sibling(div, 2);
			{
				let $0 = user_derived(() => (get(usuarioForm)?.id > 0 ? "Actualizar" : "Guardar") + " Usuario");
				let $1 = user_derived(() => get(usuarioForm)?.id > 0);
				let $2 = user_derived(() => get(usuarioForm)?.id > 0 ? () => saveUsuario(true) : void 0);
				Modal(node_1, {
					id: 1,
					size: 5,
					get title() {
						return get($0);
					},
					get isEdit() {
						return get($1);
					},
					onSave: () => saveUsuario(),
					get onDelete() {
						return get($2);
					},
					children: ($$anchor$2, $$slotProps$1) => {
						var div_4 = root_2();
						var node_2 = child(div_4);
						{
							let $0$1 = user_derived(() => get(usuarioForm)?.id > 0);
							Input(node_2, {
								save: "usuario",
								css: "col-span-12",
								label: "Usuario",
								required: true,
								get disabled() {
									return get($0$1);
								},
								get saveOn() {
									return get(usuarioForm);
								},
								set saveOn($$value) {
									set(usuarioForm, $$value, true);
								}
							});
						}
						var node_3 = sibling(node_2, 2);
						Input(node_3, {
							save: "nombres",
							css: "col-span-12",
							label: "Nombres",
							required: true,
							get saveOn() {
								return get(usuarioForm);
							},
							set saveOn($$value) {
								set(usuarioForm, $$value, true);
							}
						});
						var node_4 = sibling(node_3, 2);
						Input(node_4, {
							save: "apellidos",
							css: "col-span-12",
							label: "Apellidos",
							get saveOn() {
								return get(usuarioForm);
							},
							set saveOn($$value) {
								set(usuarioForm, $$value, true);
							}
						});
						var node_5 = sibling(node_4, 2);
						Input(node_5, {
							save: "documentoNro",
							css: "col-span-12",
							label: "Nº Documento",
							get saveOn() {
								return get(usuarioForm);
							},
							set saveOn($$value) {
								set(usuarioForm, $$value, true);
							}
						});
						var node_6 = sibling(node_5, 2);
						Input(node_6, {
							save: "cargo",
							css: "col-span-12",
							label: "Cargo",
							get saveOn() {
								return get(usuarioForm);
							},
							set saveOn($$value) {
								set(usuarioForm, $$value, true);
							}
						});
						var node_7 = sibling(node_6, 2);
						Input(node_7, {
							save: "email",
							css: "col-span-12",
							label: "Email",
							get saveOn() {
								return get(usuarioForm);
							},
							set saveOn($$value) {
								set(usuarioForm, $$value, true);
							}
						});
						var node_8 = sibling(node_7, 2);
						SearchCard(node_8, {
							save: "perfilesIDs",
							css: "col-span-24",
							get options() {
								return perfilesService.perfiles;
							},
							keyId: "id",
							keyName: "nombre",
							label: "PERFILES ::",
							get saveOn() {
								return get(usuarioForm);
							},
							set saveOn($$value) {
								set(usuarioForm, $$value, true);
							}
						});
						var node_9 = sibling(node_8, 2);
						{
							let $0$1 = user_derived(() => !get(usuarioForm).id);
							let $1$1 = user_derived(() => get(usuarioForm).id > 0 ? "SIN CAMBIAR" : "");
							Input(node_9, {
								save: "password1",
								css: "col-span-12",
								label: "Password",
								type: "password",
								get required() {
									return get($0$1);
								},
								get placeholder() {
									return get($1$1);
								},
								get saveOn() {
									return get(usuarioForm);
								},
								set saveOn($$value) {
									set(usuarioForm, $$value, true);
								}
							});
						}
						var node_10 = sibling(node_9, 2);
						{
							let $0$1 = user_derived(() => !get(usuarioForm).id);
							Input(node_10, {
								save: "password2",
								css: "col-span-12",
								label: "Password (Repetir)",
								type: "password",
								get required() {
									return get($0$1);
								},
								get saveOn() {
									return get(usuarioForm);
								},
								set saveOn($$value) {
									set(usuarioForm, $$value, true);
								}
							});
						}
						reset(div_4);
						append($$anchor$2, div_4);
					},
					$$slots: { default: true }
				});
			}
			append($$anchor$1, fragment_1);
		},
		$$slots: { default: true }
	});
	pop();
}
delegate(["keyup", "click"]);
export { _page as component };
