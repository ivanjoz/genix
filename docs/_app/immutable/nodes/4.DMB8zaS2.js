import { $ as proxy, L as delegate, M as append, P as from_html, Q as sibling, Tt as __toESM, X as child, Z as first_child, _t as reset, at as state, lt as pop, rt as set, st as user_derived, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { h as POST, i as closeModal, m as GetHandler, t as Core } from "../chunks/BwrZ3UQO.js";
import { a as throttle, o as require_notiflix_aio_3_2_8_min, t as Notify } from "../chunks/DwKmqXYU.js";
import { a as formatTime } from "../chunks/DOFkf9MZ.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as Modal } from "../chunks/iQLQpuQM.js";
var import_notiflix_aio_3_2_8_min = /* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min());
var EmpresasService = class extends GetHandler {
	route = "empresas";
	useCache = {
		min: 5,
		ver: 1
	};
	#empresas = state(proxy([]));
	get empresas() {
		return get(this.#empresas);
	}
	set empresas(value) {
		set(this.#empresas, value, true);
	}
	#empresasMap = state(proxy(/* @__PURE__ */ new Map()));
	get empresasMap() {
		return get(this.#empresasMap);
	}
	set empresasMap(value) {
		set(this.#empresasMap, value, true);
	}
	handler(response) {
		this.empresas = (response?.Records || response || []).map((empresa) => {
			empresa.SmtpConfig = empresa.SmtpConfig || {};
			empresa.CulquiConfig = empresa.CulquiConfig || {};
			return empresa;
		});
		this.empresasMap = new Map(this.empresas.map((x) => [x.id, x]));
	}
	constructor() {
		super();
		this.fetch();
	}
	updateEmpresa(empresa) {
		const existing = this.empresas.find((x) => x.id === empresa.id);
		if (existing) Object.assign(existing, empresa);
		else this.empresas.unshift(empresa);
		this.empresasMap.set(empresa.id, empresa);
	}
	removeEmpresa(id) {
		this.empresas = this.empresas.filter((x) => x.id !== id);
		this.empresasMap.delete(id);
	}
};
const postEmpresa = (data) => {
	return POST({
		data,
		route: "empresas",
		refreshRoutes: ["empresas"]
	});
};
var root_2 = from_html(`<div class="grid grid-cols-24 gap-10 p-6"><!> <!> <!> <!> <!> <!> <!> <!></div>`);
var root_1 = from_html(`<div class="h-full"><div class="flex items-center justify-between mb-6"><div class="i-search mr-16 w-256"><div><i class="icon-search"></i></div> <input class="w-full" autocomplete="off" type="text"/></div> <div class="flex items-center"><button class="bx-green" aria-label="Agregar empresa"><i class="icon-plus"></i></button></div></div> <!></div> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	const empresasService = new EmpresasService();
	let filterText = state("");
	let empresaForm = state(proxy({}));
	async function saveEmpresa(isDelete) {
		const form = get(empresaForm);
		if ((form.Nombre?.length || 0) < 3) {
			Notify.failure("El nombre de la empresa debe tener al menos 3 caracteres.");
			return;
		}
		if ((form.RUC?.length || 0) < 8) {
			Notify.failure("El RUC debe tener al menos 8 caracteres.");
			return;
		}
		if (isDelete) form.ss = 0;
		import_notiflix_aio_3_2_8_min.Loading.standard("Guardando Empresa...");
		try {
			const result = await postEmpresa(form);
			if (isDelete) empresasService.removeEmpresa(form.id);
			else {
				if (!form.id) form.id = result.id;
				empresasService.updateEmpresa(form);
			}
			closeModal(1);
			Notify.success("Empresa guardada correctamente");
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
			header: "Nombre",
			highlight: true,
			cellCss: "px-6 c-blue",
			getValue: (e) => e.Nombre
		},
		{
			header: "Razón Social",
			getValue: (e) => e.RazonSocial
		},
		{
			header: "RUC",
			headerCss: "w-120",
			cellCss: "px-6",
			getValue: (e) => e.RUC
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
				set(empresaForm, { ...rec }, true);
				Core.openModal(1);
			}
		}
	];
	Page($$anchor, {
		title: "Empresas",
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
				set(empresaForm, {
					ss: 1,
					SmtpConfig: {},
					CulquiConfig: {}
				}, true);
				Core.openModal(1);
			};
			reset(div_3);
			reset(div_1);
			VTable(sibling(div_1, 2), {
				get columns() {
					return columns;
				},
				get data() {
					return empresasService.empresas;
				},
				css: "w-full",
				maxHeight: "calc(80vh - 13rem)",
				get filterText() {
					return get(filterText);
				},
				getFilterContent: (e) => [
					e.Nombre,
					e.RazonSocial,
					e.RUC,
					e.Email
				].filter((x) => x).join(" ").toLowerCase()
			});
			reset(div);
			var node_1 = sibling(div, 2);
			{
				let $0 = user_derived(() => (get(empresaForm)?.id > 0 ? "Actualizar" : "Guardar") + " Empresa");
				let $1 = user_derived(() => get(empresaForm)?.id > 0);
				let $2 = user_derived(() => get(empresaForm)?.id > 0 ? () => saveEmpresa(true) : void 0);
				Modal(node_1, {
					id: 1,
					size: 6,
					get title() {
						return get($0);
					},
					get isEdit() {
						return get($1);
					},
					onSave: () => saveEmpresa(),
					get onDelete() {
						return get($2);
					},
					children: ($$anchor$2, $$slotProps$1) => {
						var div_4 = root_2();
						var node_2 = child(div_4);
						Input(node_2, {
							save: "Nombre",
							css: "col-span-24 md:col-span-12",
							label: "Nombre",
							required: true,
							get saveOn() {
								return get(empresaForm);
							},
							set saveOn($$value) {
								set(empresaForm, $$value, true);
							}
						});
						var node_3 = sibling(node_2, 2);
						Input(node_3, {
							save: "RazonSocial",
							css: "col-span-24 md:col-span-12",
							label: "Razón Social",
							get saveOn() {
								return get(empresaForm);
							},
							set saveOn($$value) {
								set(empresaForm, $$value, true);
							}
						});
						var node_4 = sibling(node_3, 2);
						Input(node_4, {
							save: "RUC",
							css: "col-span-24 md:col-span-8",
							label: "RUC",
							required: true,
							get saveOn() {
								return get(empresaForm);
							},
							set saveOn($$value) {
								set(empresaForm, $$value, true);
							}
						});
						var node_5 = sibling(node_4, 2);
						Input(node_5, {
							save: "Email",
							css: "col-span-24 md:col-span-8",
							label: "Email",
							type: "email",
							get saveOn() {
								return get(empresaForm);
							},
							set saveOn($$value) {
								set(empresaForm, $$value, true);
							}
						});
						var node_6 = sibling(node_5, 2);
						Input(node_6, {
							save: "Telefono",
							css: "col-span-24 md:col-span-8",
							label: "Teléfono",
							get saveOn() {
								return get(empresaForm);
							},
							set saveOn($$value) {
								set(empresaForm, $$value, true);
							}
						});
						var node_7 = sibling(node_6, 2);
						Input(node_7, {
							save: "Representante",
							css: "col-span-24 md:col-span-12",
							label: "Representante",
							get saveOn() {
								return get(empresaForm);
							},
							set saveOn($$value) {
								set(empresaForm, $$value, true);
							}
						});
						var node_8 = sibling(node_7, 2);
						Input(node_8, {
							save: "Ciudad",
							css: "col-span-24 md:col-span-12",
							label: "Ciudad",
							get saveOn() {
								return get(empresaForm);
							},
							set saveOn($$value) {
								set(empresaForm, $$value, true);
							}
						});
						Input(sibling(node_8, 2), {
							save: "Direccion",
							css: "col-span-24",
							label: "Dirección",
							useTextArea: true,
							rows: 2,
							get saveOn() {
								return get(empresaForm);
							},
							set saveOn($$value) {
								set(empresaForm, $$value, true);
							}
						});
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
