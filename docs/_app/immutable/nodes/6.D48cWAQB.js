import { $ as proxy, A as if_block, C as clsx, L as delegate, M as append, N as comment, O as each, P as from_html, Q as sibling, S as set_class, Tt as __toESM, W as template_effect, X as child, Z as first_child, _t as reset, a as prop, at as state, j as set_text, k as index, lt as pop, rt as set, st as user_derived, ut as push, x as set_style, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { h as POST, i as closeModal, m as GetHandler, t as Core } from "../chunks/BwrZ3UQO.js";
import { a as throttle, o as require_notiflix_aio_3_2_8_min, t as Notify } from "../chunks/DwKmqXYU.js";
import "../chunks/DOFkf9MZ.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
import "../chunks/CGFlWCTf.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as modules_default } from "../chunks/CoH7Ebne.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as Modal } from "../chunks/iQLQpuQM.js";
import "../chunks/C1AypaHL.js";
import { t as SearchSelect } from "../chunks/BOmyLNB5.js";
import { t as SearchCard } from "../chunks/BHNEJas0.js";
import { t as CheckboxOptions } from "../chunks/Bmy_fcV9.js";
var import_notiflix_aio_3_2_8_min = /* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min());
const accesosGrupos = [
	{
		id: 1,
		name: "Gestión"
	},
	{
		id: 2,
		name: "Seguridad"
	},
	{
		id: 3,
		name: "Maestros"
	},
	{
		id: 4,
		name: "Productos"
	},
	{
		id: 5,
		name: "Reportes"
	}
];
const accesoAcciones = [
	{
		id: 1,
		name: "Visualizar",
		short: "VER",
		icon: "icon-eye",
		color: "#00c07d",
		color2: "#49c99c"
	},
	{
		id: 2,
		name: "Editar",
		short: "EDITAR",
		icon: "icon-pencil",
		color: "#0080f9"
	},
	{
		id: 3,
		name: "Eliminar",
		short: "ELIMINAR",
		icon: "icon-trash",
		color: "#0080f9"
	},
	{
		id: 7,
		name: "Todo",
		short: "EDITAR",
		icon: "icon-shield",
		color: "#af12eb",
		color2: "#d35eff"
	}
];
var AccesosService = class extends GetHandler {
	route = "seguridad-accesos";
	useCache = {
		min: 5,
		ver: 1
	};
	#accesos = state(proxy([]));
	get accesos() {
		return get(this.#accesos);
	}
	set accesos(value) {
		set(this.#accesos, value, true);
	}
	#accesosMap = state(proxy(/* @__PURE__ */ new Map()));
	get accesosMap() {
		return get(this.#accesosMap);
	}
	set accesosMap(value) {
		set(this.#accesosMap, value, true);
	}
	handler(response) {
		this.accesos = response || [];
		this.accesosMap = new Map(this.accesos.map((x) => [x.id, x]));
	}
	constructor() {
		super();
		this.fetch();
	}
	updateAcceso(acceso) {
		const existing = this.accesos.find((x) => x.id === acceso.id);
		if (existing) Object.assign(existing, acceso);
		else this.accesos.unshift(acceso);
		this.accesosMap.set(acceso.id, acceso);
	}
	removeAcceso(id) {
		this.accesos = this.accesos.filter((x) => x.id !== id);
		this.accesosMap.delete(id);
	}
};
var PerfilesService = class extends GetHandler {
	route = "perfiles";
	useCache = {
		min: 5,
		ver: 1
	};
	#perfiles = state(proxy([]));
	get perfiles() {
		return get(this.#perfiles);
	}
	set perfiles(value) {
		set(this.#perfiles, value, true);
	}
	#perfilesMap = state(proxy(/* @__PURE__ */ new Map()));
	get perfilesMap() {
		return get(this.#perfilesMap);
	}
	set perfilesMap(value) {
		set(this.#perfilesMap, value, true);
	}
	handler(response) {
		const perfiles = (response || []).filter((x) => x.ss > 0);
		for (const pr of perfiles) {
			pr.accesos = pr.accesos || [];
			pr.accesosMap = pr.accesosMap || /* @__PURE__ */ new Map();
			for (const e of pr.accesos) {
				const id = Math.floor(e / 10);
				const nivel = e - id * 10;
				pr.accesosMap.has(id) ? pr.accesosMap.get(id).push(nivel) : pr.accesosMap.set(id, [nivel]);
			}
		}
		this.perfiles = perfiles;
		this.perfilesMap = new Map(perfiles.map((x) => [x.id, x]));
	}
	constructor() {
		super();
		this.fetch();
	}
	updatePerfil(perfil) {
		const existing = this.perfiles.find((x) => x.id === perfil.id);
		if (existing) Object.assign(existing, perfil);
		else this.perfiles.unshift(perfil);
		this.perfilesMap.set(perfil.id, perfil);
	}
	removePerfil(id) {
		this.perfiles = this.perfiles.filter((x) => x.id !== id);
		this.perfilesMap.delete(id);
	}
};
const postSeguridadAccesos = (data) => {
	return POST({
		data,
		route: "seguridad/accesos",
		refreshRoutes: ["seguridad/accesos"]
	});
};
const postPerfil = (data) => {
	const dataToSend = { ...data };
	delete dataToSend.accesosMap;
	delete dataToSend._open;
	return POST({
		data: dataToSend,
		route: "perfiles",
		refreshRoutes: ["perfiles"]
	});
};
function arrayToMapN(arr, key) {
	return new Map(arr.map((item) => [item[key], item]));
}
var root_1$1 = from_html(`<div class="aLf p-abs aMr"></div>`);
var root_2$1 = from_html(`<div class="absolute flex items-center justify-center aLb aMr" role="button" tabindex="0"><i class="icon-pencil"></i></div> <div class="absolute flex items-center justify-center aLc c-purple aMr"> </div>`, 1);
var root_5$1 = from_html(`<div class="aLg aMr"><i></i></div>`);
var root_7$1 = from_html(`<div class="accion-btn aMr" role="button" tabindex="0"> </div>`);
var root_3$1 = from_html(`<div class="aLe aMr"></div> <div class="aLa w-full flex justify-center z-10 aMr"></div>`, 1);
var root = from_html(`<div role="button" tabindex="0"><div class="mr-4 aMr"> </div> <!> <!> <!></div>`);
function AccesoCard($$anchor, $$props) {
	push($$props, true);
	const accesoAccionesMap = arrayToMapN(accesoAcciones, "id");
	let perfilForm = prop($$props, "perfilForm", 15);
	const acciones = user_derived(() => perfilForm()?.accesosMap?.get($$props.acceso.id) || []);
	const accionColor = user_derived(() => {
		if (get(acciones).length === 0 || $$props.isEdit) return void 0;
		const acciones_ = [...get(acciones)].sort().reverse();
		const accion = accesoAccionesMap.get(acciones_[0] || 0);
		return accion?.color2 || accion?.color || "";
	});
	function handleCardClick(ev) {
		if (Core.deviceType === 1 || !perfilForm()) return;
		ev.stopPropagation();
		const currentAcciones = perfilForm().accesosMap.get($$props.acceso.id) || [];
		if (currentAcciones.length >= $$props.acceso.acciones.length) perfilForm().accesosMap.delete($$props.acceso.id);
		else {
			const missing = $$props.acceso.acciones.filter((x) => !currentAcciones.includes(x));
			const newAcciones = [...currentAcciones, missing[0]].filter((x) => x);
			perfilForm().accesosMap.set($$props.acceso.id, newAcciones);
		}
		perfilForm(perfilForm().accesosMap = new Map(perfilForm().accesosMap), true);
	}
	function handleAccionClick(ev, id) {
		ev.stopPropagation();
		if (!perfilForm()) return;
		let newAcciones = [...perfilForm().accesosMap.get($$props.acceso.id) || []];
		if (newAcciones.includes(id)) newAcciones = newAcciones.filter((x) => x !== id);
		else newAcciones.push(id);
		newAcciones.sort((a, b) => b - a);
		if (newAcciones.length === 0) perfilForm().accesosMap.delete($$props.acceso.id);
		else perfilForm().accesosMap.set($$props.acceso.id, newAcciones);
		perfilForm(perfilForm().accesosMap = new Map(perfilForm().accesosMap), true);
	}
	const cN = user_derived(() => {
		let className = "acceso-card";
		if ($$props.isEdit) className += " sel";
		if (Core.deviceType > 1) className += " mobile";
		return className;
	});
	var div = root();
	div.__click = handleCardClick;
	div.__keydown = (ev) => {
		if (ev.key === "Enter" || ev.key === " ") {
			ev.preventDefault();
			handleCardClick(ev);
		}
	};
	let styles;
	var div_1 = child(div);
	var text = child(div_1, true);
	reset(div_1);
	var node = sibling(div_1, 2);
	var consequent = ($$anchor$1) => {
		var div_2 = root_1$1();
		let styles_1;
		template_effect(() => styles_1 = set_style(div_2, "", styles_1, { "background-color": get(accionColor) }));
		append($$anchor$1, div_2);
	};
	if_block(node, ($$render) => {
		if (Core.deviceType > 1) $$render(consequent);
	});
	var node_1 = sibling(node, 2);
	var consequent_1 = ($$anchor$1) => {
		var fragment = root_2$1();
		var div_3 = first_child(fragment);
		div_3.__click = (ev) => {
			ev.stopPropagation();
			$$props.onEdit();
		};
		div_3.__keydown = (ev) => {
			if (ev.key === "Enter" || ev.key === " ") {
				ev.preventDefault();
				$$props.onEdit();
			}
		};
		var div_4 = sibling(div_3, 2);
		var text_1 = child(div_4, true);
		reset(div_4);
		template_effect(() => set_text(text_1, $$props.acceso.id));
		append($$anchor$1, fragment);
	};
	if_block(node_1, ($$render) => {
		if ($$props.isEdit) $$render(consequent_1);
	});
	var node_2 = sibling(node_1, 2);
	var consequent_4 = ($$anchor$1) => {
		var fragment_1 = root_3$1();
		var div_5 = first_child(fragment_1);
		each(div_5, 21, () => get(acciones), index, ($$anchor$2, id) => {
			const accion = user_derived(() => accesoAccionesMap.get(get(id)));
			var fragment_2 = comment();
			var node_3 = first_child(fragment_2);
			var consequent_2 = ($$anchor$3) => {
				var div_6 = root_5$1();
				let styles_2;
				var i = child(div_6);
				reset(div_6);
				template_effect(() => {
					styles_2 = set_style(div_6, "", styles_2, { "background-color": get(accion).color });
					set_class(i, 1, clsx(get(accion).icon), "aMr");
				});
				append($$anchor$3, div_6);
			};
			if_block(node_3, ($$render) => {
				if (get(accion)) $$render(consequent_2);
			});
			append($$anchor$2, fragment_2);
		});
		reset(div_5);
		var div_7 = sibling(div_5, 2);
		each(div_7, 21, () => $$props.acceso.acciones, index, ($$anchor$2, id) => {
			const accion = user_derived(() => accesoAccionesMap.get(get(id)));
			const selected = user_derived(() => get(acciones).includes(get(id)));
			var fragment_3 = comment();
			var node_4 = first_child(fragment_3);
			var consequent_3 = ($$anchor$3) => {
				var div_8 = root_7$1();
				div_8.__click = (ev) => handleAccionClick(ev, get(id));
				div_8.__keydown = (ev) => {
					if (ev.key === "Enter" || ev.key === " ") {
						ev.preventDefault();
						handleAccionClick(ev, get(id));
					}
				};
				let styles_3;
				var text_2 = child(div_8, true);
				reset(div_8);
				template_effect(() => {
					styles_3 = set_style(div_8, "", styles_3, {
						"background-color": get(selected) ? get(accion).color : void 0,
						"border-color": get(selected) ? get(accion).color : void 0,
						color: get(selected) ? "white" : void 0
					});
					set_text(text_2, get(accion).short);
				});
				append($$anchor$3, div_8);
			};
			if_block(node_4, ($$render) => {
				if (get(accion)) $$render(consequent_3);
			});
			append($$anchor$2, fragment_3);
		});
		reset(div_7);
		append($$anchor$1, fragment_1);
	};
	if_block(node_2, ($$render) => {
		if (!$$props.isEdit && perfilForm()) $$render(consequent_4);
	});
	reset(div);
	template_effect(() => {
		set_class(div, 1, clsx(get(cN)), "aMr");
		styles = set_style(div, "", styles, { "border-left-color": get(accionColor) });
		set_text(text, $$props.acceso.nombre);
	});
	append($$anchor, div);
	pop();
}
delegate(["click", "keydown"]);
var root_2 = from_html(`<span class="c-red">(Modo Edición)</span>`);
var root_3 = from_html(`<span class="mr-4">:</span> <span class="c-purple ml-4"> </span>`, 1);
var root_4 = from_html(`<button class="bx-green mr-8" aria-label="Agregar acceso"><i class="icon-plus"></i></button>`);
var root_5 = from_html(`<button class="bx-blue mr-8" aria-label="Guardar"><i class="icon-floppy"></i> <span class="max-md:hidden">Guardar</span></button>`);
var root_7 = from_html(`<i class="c-red icon-cancel"></i>`);
var root_8 = from_html(`<i class="icon-pencil"></i>`);
var root_6 = from_html(`<button class="bn-white" aria-label="Editar"><!></button>`);
var root_9 = from_html(`<div class="mb-8 px-12 py-8 bg-red-100 border border-red-400 text-red-700 rounded w-fit">Debe seleccionar un perfil para editar sus accesos.</div>`);
var root_10 = from_html(`<div class="ff-bold h3 mb-6"> </div> <div class="grid grid-cols-3 gap-x-12 gap-y-8 mb-16"></div>`, 1);
var root_12 = from_html(`<div class="grid grid-cols-24 gap-10 p-6"><!> <!> <!> <!> <!> <div class="col-span-24 flex items-center gap-12 mb-4"><div class="h-[1px] grow bg-gray-300"></div> <div class="ff-bold text-lg">Acciones</div> <div class="h-[1px] grow bg-gray-300"></div></div> <!></div>`);
var root_13 = from_html(`<div class="grid grid-cols-24 gap-10 p-6"><!> <!></div>`);
var root_1 = from_html(`<div class="flex justify-between h-full gap-8 max-md:flex-col"><div class="w-full md:w-[32%]"><div class="flex justify-between items-center w-full mb-10"><div class="i-search mr-16 w-256"><div><i class="icon-search"></i></div> <input class="w-full" autocomplete="off" type="text"/></div> <div class="flex items-center"><button class="bx-green" aria-label="Agregar perfil"><i class="icon-plus"></i></button></div></div> <!></div> <div class="w-full md:w-[66.5%]"><div class="flex justify-between w-full mb-6"><div class="ff-bold text-xl"><span>Accesos</span> <!> <!></div> <div class="flex items-center max-md:absolute max-md:top-0 max-md:right-0"><!> <!> <!></div></div> <!> <!></div></div> <!> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	const accesosService = new AccesosService();
	const perfilesService = new PerfilesService();
	const accesosGruposMap = arrayToMapN(accesosGrupos, "id");
	const modulesMap = arrayToMapN(modules_default, "id");
	let accesoForm = state(proxy({}));
	let perfilForm = state(proxy({}));
	let filterText = state("");
	let accesoEdit = state(false);
	const accesosGrouped = user_derived(() => {
		const gruposMap = /* @__PURE__ */ new Map();
		for (let acs of accesosService.accesos) for (let md of acs.modulosIDs) {
			const key = [md, acs.grupo].join("_");
			if (!gruposMap.has(key)) gruposMap.set(key, []);
			gruposMap.get(key).push(acs);
			break;
		}
		const accesosGrouped_ = [];
		for (let [key, accesosGroup] of gruposMap) {
			const [moduleID, group] = key.split("_").map((x) => parseInt(x));
			accesosGrouped_.push({
				moduleID,
				group,
				accesos: accesosGroup,
				groupName: accesosGruposMap.get(group)?.name || "",
				moduleName: modulesMap.get(moduleID)?.name || ""
			});
		}
		return accesosGrouped_;
	});
	async function saveAcceso() {
		const form = get(accesoForm);
		if (!form.nombre || !form.orden || (form.acciones?.length || 0) == 0 || !form.grupo || (form.modulosIDs?.length || 0) === 0) {
			Notify.failure("Faltan propiedades para agregar el acceso.");
			return;
		}
		import_notiflix_aio_3_2_8_min.Loading.standard("Actualizando Acceso...");
		try {
			const result = await postSeguridadAccesos(form);
			if ((form.id || 0) <= 0) form.id = result.id;
			accesosService.updateAcceso(form);
			set(accesoForm, {}, true);
			closeModal(1);
			Notify.success("Acceso guardado correctamente");
		} catch (error) {
			Notify.failure(error);
		}
		import_notiflix_aio_3_2_8_min.Loading.remove();
	}
	async function savePerfil(onDelete, isAccesos) {
		const form = get(perfilForm);
		if (!form.nombre) {
			Notify.failure("Faltan propiedades para agregar el perfil.");
			return;
		}
		if (isAccesos) {
			form.accesos = [];
			for (let [accesoID, niveles] of form.accesosMap) {
				if (niveles.length === 0) form.accesosMap.delete(accesoID);
				for (let n of niveles) form.accesos.push(accesoID * 10 + n);
			}
			const accesosFiltered = accesosService.accesos.filter((x) => form.accesosMap.has(x.id));
			const modulosIDSet = /* @__PURE__ */ new Set();
			for (let e of accesosFiltered) for (let md of e.modulosIDs) modulosIDSet.add(md);
			form.modulosIDs = [...modulosIDSet];
		}
		import_notiflix_aio_3_2_8_min.Loading.standard("Actualizando Perfil...");
		try {
			const result = await postPerfil(form);
			if ((form.id || 0) <= 0) form.id = result.id;
			form._open = false;
			perfilesService.updatePerfil(form);
			set(perfilForm, {}, true);
			closeModal(2);
			Notify.success("Perfil guardado correctamente");
		} catch (error) {
			Notify.failure(error);
		}
		import_notiflix_aio_3_2_8_min.Loading.remove();
	}
	const columns = [
		{
			header: "ID",
			headerCss: "w-54",
			cellCss: "text-center c-purple",
			getValue: (e) => e.id
		},
		{
			header: "Perfil",
			highlight: true,
			getValue: (e) => e.nombre
		},
		{
			header: "...",
			headerCss: "w-42",
			cellCss: "text-center",
			id: "actions",
			buttonEditHandler: (rec) => {
				set(perfilForm, {
					...rec,
					accesosMap: new Map(rec.accesosMap)
				}, true);
				set(accesoEdit, false);
				Core.openModal(2);
			}
		}
	];
	Page($$anchor, {
		title: "Perfiles & Accesos",
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = first_child(fragment_1);
			var div_1 = child(div);
			var div_2 = child(div_1);
			var div_3 = child(div_2);
			var input = sibling(child(div_3), 2);
			input.__keyup = (ev) => {
				ev.stopPropagation();
				throttle(() => {
					set(filterText, (ev.target.value || "").toLowerCase().trim(), true);
				}, 150);
			};
			reset(div_3);
			var div_4 = sibling(div_3, 2);
			var button = child(div_4);
			button.__click = (ev) => {
				ev.stopPropagation();
				set(perfilForm, {
					ss: 1,
					accesosMap: /* @__PURE__ */ new Map()
				}, true);
				Core.openModal(2);
			};
			reset(div_4);
			reset(div_2);
			VTable(sibling(div_2, 2), {
				css: "w-full selectable",
				get columns() {
					return columns;
				},
				maxHeight: "calc(100vh - 8rem - 16px)",
				get data() {
					return perfilesService.perfiles;
				},
				get selected() {
					return get(perfilForm).id;
				},
				get filterText() {
					return get(filterText);
				},
				getFilterContent: (e) => [e.nombre].filter((x) => x).join(" ").toLowerCase(),
				isSelected: (e, id) => e.id === id,
				onRowClick: (e) => {
					set(accesoEdit, false);
					if (e.id === get(perfilForm).id) set(perfilForm, {}, true);
					else set(perfilForm, {
						...e,
						_open: true,
						accesosMap: new Map(e.accesosMap)
					}, true);
				}
			});
			reset(div_1);
			var div_5 = sibling(div_1, 2);
			var div_6 = child(div_5);
			var div_7 = child(div_6);
			var node_1 = sibling(child(div_7), 2);
			var consequent = ($$anchor$2) => {
				append($$anchor$2, root_2());
			};
			if_block(node_1, ($$render) => {
				if (get(accesoEdit)) $$render(consequent);
			});
			var node_2 = sibling(node_1, 2);
			var consequent_1 = ($$anchor$2) => {
				var fragment_2 = root_3();
				var span_1 = sibling(first_child(fragment_2), 2);
				var text = child(span_1, true);
				reset(span_1);
				template_effect(() => set_text(text, get(perfilForm).nombre));
				append($$anchor$2, fragment_2);
			};
			if_block(node_2, ($$render) => {
				if (get(perfilForm).id > 0) $$render(consequent_1);
			});
			reset(div_7);
			var div_8 = sibling(div_7, 2);
			var node_3 = child(div_8);
			var consequent_2 = ($$anchor$2) => {
				var button_1 = root_4();
				button_1.__click = (ev) => {
					ev.stopPropagation();
					set(accesoForm, {
						ss: 1,
						modulosIDs: [],
						acciones: [],
						id: 0,
						nombre: "",
						orden: 0,
						grupo: 0,
						upd: 0
					}, true);
					Core.openModal(1);
				};
				append($$anchor$2, button_1);
			};
			if_block(node_3, ($$render) => {
				if (get(accesoEdit)) $$render(consequent_2);
			});
			var node_4 = sibling(node_3, 2);
			var consequent_3 = ($$anchor$2) => {
				var button_2 = root_5();
				button_2.__click = (ev) => {
					ev.stopPropagation();
					savePerfil(false, true);
				};
				append($$anchor$2, button_2);
			};
			if_block(node_4, ($$render) => {
				if (get(perfilForm).id > 0) $$render(consequent_3);
			});
			var node_5 = sibling(node_4, 2);
			var consequent_5 = ($$anchor$2) => {
				var button_3 = root_6();
				button_3.__click = (ev) => {
					ev.stopPropagation();
					set(accesoEdit, !get(accesoEdit));
				};
				var node_6 = child(button_3);
				var consequent_4 = ($$anchor$3) => {
					append($$anchor$3, root_7());
				};
				var alternate = ($$anchor$3) => {
					append($$anchor$3, root_8());
				};
				if_block(node_6, ($$render) => {
					if (get(accesoEdit)) $$render(consequent_4);
					else $$render(alternate, false);
				});
				reset(button_3);
				append($$anchor$2, button_3);
			};
			if_block(node_5, ($$render) => {
				if (!get(perfilForm).id) $$render(consequent_5);
			});
			reset(div_8);
			reset(div_6);
			var node_7 = sibling(div_6, 2);
			var consequent_6 = ($$anchor$2) => {
				append($$anchor$2, root_9());
			};
			if_block(node_7, ($$render) => {
				if (!get(accesoEdit) && !get(perfilForm).id) $$render(consequent_6);
			});
			each(sibling(node_7, 2), 17, () => get(accesosGrouped), index, ($$anchor$2, ag) => {
				var fragment_3 = root_10();
				var div_10 = first_child(fragment_3);
				var text_1 = child(div_10);
				reset(div_10);
				var div_11 = sibling(div_10, 2);
				each(div_11, 21, () => get(ag).accesos, index, ($$anchor$3, acceso) => {
					AccesoCard($$anchor$3, {
						get acceso() {
							return get(acceso);
						},
						get isEdit() {
							return get(accesoEdit);
						},
						onEdit: () => {
							set(accesoForm, { ...get(acceso) }, true);
							Core.openModal(1);
						},
						get perfilForm() {
							return get(perfilForm);
						},
						set perfilForm($$value) {
							set(perfilForm, $$value, true);
						}
					});
				});
				reset(div_11);
				template_effect(() => set_text(text_1, `${get(ag).moduleName ?? ""}${get(ag).moduleName ? " > " : ""}${get(ag).groupName ?? ""}`));
				append($$anchor$2, fragment_3);
			});
			reset(div_5);
			reset(div);
			var node_9 = sibling(div, 2);
			{
				let $0 = user_derived(() => (get(accesoForm)?.id > 0 ? "Editando" : "Creando") + " Acceso");
				let $1 = user_derived(() => !!get(accesoForm)?.id);
				Modal(node_9, {
					id: 1,
					size: 6,
					get title() {
						return get($0);
					},
					get isEdit() {
						return get($1);
					},
					onSave: () => saveAcceso(),
					children: ($$anchor$2, $$slotProps$1) => {
						var div_12 = root_12();
						var node_10 = child(div_12);
						Input(node_10, {
							save: "nombre",
							css: "col-span-14",
							label: "Nombre",
							required: true,
							get saveOn() {
								return get(accesoForm);
							},
							set saveOn($$value) {
								set(accesoForm, $$value, true);
							}
						});
						var node_11 = sibling(node_10, 2);
						SearchSelect(node_11, {
							save: "grupo",
							css: "col-span-10",
							label: "Grupo",
							required: true,
							get options() {
								return accesosGrupos;
							},
							keyId: "id",
							keyName: "name",
							get saveOn() {
								return get(accesoForm);
							},
							set saveOn($$value) {
								set(accesoForm, $$value, true);
							}
						});
						var node_12 = sibling(node_11, 2);
						Input(node_12, {
							save: "descripcion",
							css: "col-span-16",
							label: "Descripción",
							get saveOn() {
								return get(accesoForm);
							},
							set saveOn($$value) {
								set(accesoForm, $$value, true);
							}
						});
						var node_13 = sibling(node_12, 2);
						Input(node_13, {
							save: "orden",
							css: "col-span-8",
							label: "Orden",
							type: "number",
							get saveOn() {
								return get(accesoForm);
							},
							set saveOn($$value) {
								set(accesoForm, $$value, true);
							}
						});
						var node_14 = sibling(node_13, 2);
						SearchCard(node_14, {
							save: "modulosIDs",
							css: "col-span-24",
							label: "Módulos",
							get options() {
								return modules_default;
							},
							keyId: "id",
							keyName: "name",
							get saveOn() {
								return get(accesoForm);
							},
							set saveOn($$value) {
								set(accesoForm, $$value, true);
							}
						});
						CheckboxOptions(sibling(node_14, 4), {
							save: "acciones",
							css: "col-span-24 flex-wrap gap-y-8",
							get options() {
								return accesoAcciones;
							},
							keyId: "id",
							keyName: "name",
							type: "multiple",
							get saveOn() {
								return get(accesoForm);
							},
							set saveOn($$value) {
								set(accesoForm, $$value, true);
							}
						});
						reset(div_12);
						append($$anchor$2, div_12);
					},
					$$slots: { default: true }
				});
			}
			var node_16 = sibling(node_9, 2);
			{
				let $0 = user_derived(() => (get(perfilForm)?.id > 0 ? "Editando" : "Creando") + " Perfil");
				let $1 = user_derived(() => !!get(perfilForm)?.id);
				Modal(node_16, {
					id: 2,
					size: 5,
					get title() {
						return get($0);
					},
					get isEdit() {
						return get($1);
					},
					onSave: () => savePerfil(),
					onClose: () => {
						set(perfilForm, {}, true);
					},
					children: ($$anchor$2, $$slotProps$1) => {
						var div_13 = root_13();
						var node_17 = child(div_13);
						Input(node_17, {
							save: "nombre",
							css: "col-span-24",
							label: "Nombre",
							required: true,
							get saveOn() {
								return get(perfilForm);
							},
							set saveOn($$value) {
								set(perfilForm, $$value, true);
							}
						});
						Input(sibling(node_17, 2), {
							save: "descripcion",
							css: "col-span-24",
							label: "Descripción",
							get saveOn() {
								return get(perfilForm);
							},
							set saveOn($$value) {
								set(perfilForm, $$value, true);
							}
						});
						reset(div_13);
						append($$anchor$2, div_13);
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
