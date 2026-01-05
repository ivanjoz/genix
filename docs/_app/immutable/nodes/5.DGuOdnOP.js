import { $ as proxy, L as delegate, M as append, P as from_html, Q as sibling, Tt as __toESM, X as child, Z as first_child, _t as reset, at as state, lt as pop, nt as mutate, rt as set, tt as mutable_source, u as init, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { h as POST, m as GetHandler } from "../chunks/BwrZ3UQO.js";
import { o as require_notiflix_aio_3_2_8_min, t as Notify } from "../chunks/DwKmqXYU.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import "../chunks/BGbcPCgU.js";
var import_notiflix_aio_3_2_8_min = /* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min());
var EmpresaParametrosService = class extends GetHandler {
	route = "empresa-parametros";
	useCache = {
		min: 10,
		ver: 1
	};
	#empresa = state(proxy({
		SmtpConfig: {},
		CulquiConfig: {}
	}));
	get empresa() {
		return get(this.#empresa);
	}
	set empresa(value) {
		set(this.#empresa, value, true);
	}
	handler(response) {
		const record = response[0] || {};
		console.log("empresa response::", record);
		record.SmtpConfig = record.SmtpConfig || {};
		record.CulquiConfig = record.CulquiConfig || {};
		this.empresa = record;
	}
	constructor() {
		super();
		this.fetch();
	}
};
const postEmpresaParametros = (data) => {
	return POST({
		data,
		route: "empresa-parametros",
		refreshRoutes: ["empresa-parametros"]
	});
};
var root_1 = from_html(`<div class="flex justify-between items-center mb-8"><div><div class="h3 ff-bold">Parámetros de la Empresa</div></div> <div class="flex items-center"><button class="bx-blue"><i class="icon-floppy mr-2"></i>Guardar</button></div></div> <div class="w-full grid grid-cols-1 lg:grid-cols-[4fr_3fr] gap-4"><div class="grid grid-cols-24 gap-10 content-start"><!> <!> <!> <!> <!> <!> <!> <!> <div class="col-span-24 mb-2"><div class="h3 ff-bold">Ecommerce</div></div> <!> <!> <!> <!> <!> <!></div> <div style="margin-left: 16px"><div class="w-full py-10 px-12 bg-white rounded shadow-sm"><div class="w-full mb-8">Parámetros SMTP para notificaciones</div> <div class="grid grid-cols-24 gap-4"><!> <!> <!> <!> <!></div></div></div></div>`, 1);
function _page($$anchor, $$props) {
	push($$props, false);
	const service = mutable_source(new EmpresaParametrosService());
	async function saveEmpresa() {
		const form = get(service).empresa;
		if ((form.RUC || "").length === 0 || (form.Nombre || "").length === 0 || (form.RazonSocial || "").length === 0 || (form.Email || "").length === 0) {
			Notify.failure("Faltan datos a guardar.");
			return;
		}
		import_notiflix_aio_3_2_8_min.Loading.standard("Guardando...");
		try {
			await postEmpresaParametros(form);
			Notify.success("Datos guardados correctamente");
		} catch (error) {}
		import_notiflix_aio_3_2_8_min.Loading.remove();
	}
	init();
	Page($$anchor, {
		title: "Parámetros",
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = first_child(fragment_1);
			var div_1 = sibling(child(div), 2);
			var button = child(div_1);
			button.__click = (ev) => {
				ev.stopPropagation();
				saveEmpresa();
			};
			reset(div_1);
			reset(div);
			var div_2 = sibling(div, 2);
			var div_3 = child(div_2);
			var node = child(div_3);
			Input(node, {
				css: "col-span-14",
				label: "Nombre",
				save: "Nombre",
				required: true,
				get saveOn() {
					return get(service).empresa;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa = $$value);
				},
				$$legacy: true
			});
			var node_1 = sibling(node, 2);
			Input(node_1, {
				css: "col-span-10",
				label: "RUC",
				save: "RUC",
				required: true,
				get saveOn() {
					return get(service).empresa;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa = $$value);
				},
				$$legacy: true
			});
			var node_2 = sibling(node_1, 2);
			Input(node_2, {
				css: "col-span-14",
				label: "Razón Social",
				save: "RazonSocial",
				required: true,
				get saveOn() {
					return get(service).empresa;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa = $$value);
				},
				$$legacy: true
			});
			var node_3 = sibling(node_2, 2);
			Input(node_3, {
				css: "col-span-10",
				label: "Teléfono",
				save: "Telefono",
				get saveOn() {
					return get(service).empresa;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa = $$value);
				},
				$$legacy: true
			});
			var node_4 = sibling(node_3, 2);
			Input(node_4, {
				css: "col-span-14",
				label: "Correo Electrónico",
				save: "Email",
				required: true,
				get saveOn() {
					return get(service).empresa;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa = $$value);
				},
				$$legacy: true
			});
			var node_5 = sibling(node_4, 2);
			Input(node_5, {
				css: "col-span-10",
				label: "Representante",
				save: "Representante",
				get saveOn() {
					return get(service).empresa;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa = $$value);
				},
				$$legacy: true
			});
			var node_6 = sibling(node_5, 2);
			Input(node_6, {
				css: "col-span-14",
				label: "Dirección Legal",
				save: "Direccion",
				get saveOn() {
					return get(service).empresa;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa = $$value);
				},
				$$legacy: true
			});
			var node_7 = sibling(node_6, 2);
			Input(node_7, {
				css: "col-span-10",
				label: "Ciudad",
				save: "Ciudad",
				get saveOn() {
					return get(service).empresa;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa = $$value);
				},
				$$legacy: true
			});
			var node_8 = sibling(node_7, 4);
			Input(node_8, {
				css: "col-span-12",
				label: "Llave Pública (Pruebas)",
				save: "LlavePubDev",
				get saveOn() {
					return get(service).empresa.CulquiConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.CulquiConfig = $$value);
				},
				$$legacy: true
			});
			var node_9 = sibling(node_8, 2);
			Input(node_9, {
				css: "col-span-12",
				label: "Llave Privada (Pruebas)",
				save: "LlaveDev",
				get saveOn() {
					return get(service).empresa.CulquiConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.CulquiConfig = $$value);
				},
				$$legacy: true
			});
			var node_10 = sibling(node_9, 2);
			Input(node_10, {
				css: "col-span-12",
				label: "Llave Pública (Live)",
				save: "LlavePubLive",
				get saveOn() {
					return get(service).empresa.CulquiConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.CulquiConfig = $$value);
				},
				$$legacy: true
			});
			var node_11 = sibling(node_10, 2);
			Input(node_11, {
				css: "col-span-12",
				label: "Llave Privada (Live)",
				save: "LlaveLive",
				get saveOn() {
					return get(service).empresa.CulquiConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.CulquiConfig = $$value);
				},
				$$legacy: true
			});
			var node_12 = sibling(node_11, 2);
			Input(node_12, {
				css: "col-span-12",
				label: "Culqui LLave RSA ID",
				save: "RsaKeyID",
				get saveOn() {
					return get(service).empresa.CulquiConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.CulquiConfig = $$value);
				},
				$$legacy: true
			});
			Input(sibling(node_12, 2), {
				css: "col-span-12",
				label: "Culqui LLave RSA",
				save: "RsaKey",
				get saveOn() {
					return get(service).empresa.CulquiConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.CulquiConfig = $$value);
				},
				$$legacy: true
			});
			reset(div_3);
			var div_4 = sibling(div_3, 2);
			var div_5 = child(div_4);
			var div_6 = sibling(child(div_5), 2);
			var node_14 = child(div_6);
			Input(node_14, {
				css: "col-span-12",
				label: "Host",
				save: "Host",
				get saveOn() {
					return get(service).empresa.SmtpConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.SmtpConfig = $$value);
				},
				$$legacy: true
			});
			var node_15 = sibling(node_14, 2);
			Input(node_15, {
				css: "col-span-12",
				label: "Port",
				save: "Port",
				type: "number",
				get saveOn() {
					return get(service).empresa.SmtpConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.SmtpConfig = $$value);
				},
				$$legacy: true
			});
			var node_16 = sibling(node_15, 2);
			Input(node_16, {
				css: "col-span-12",
				label: "Usuario",
				save: "User",
				get saveOn() {
					return get(service).empresa.SmtpConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.SmtpConfig = $$value);
				},
				$$legacy: true
			});
			var node_17 = sibling(node_16, 2);
			Input(node_17, {
				css: "col-span-12",
				label: "Password",
				save: "Password",
				get saveOn() {
					return get(service).empresa.SmtpConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.SmtpConfig = $$value);
				},
				$$legacy: true
			});
			Input(sibling(node_17, 2), {
				css: "col-span-24",
				label: "Email",
				save: "Email",
				get saveOn() {
					return get(service).empresa.SmtpConfig;
				},
				set saveOn($$value) {
					mutate(service, get(service).empresa.SmtpConfig = $$value);
				},
				$$legacy: true
			});
			reset(div_6);
			reset(div_5);
			reset(div_4);
			reset(div_2);
			append($$anchor$1, fragment_1);
		},
		$$slots: { default: true }
	});
	pop();
}
delegate(["click"]);
export { _page as component };
