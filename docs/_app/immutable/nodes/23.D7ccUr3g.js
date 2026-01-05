import { $ as proxy, A as if_block, L as delegate, M as append, N as comment, O as each, P as from_html, Q as sibling, W as template_effect, X as child, Z as first_child, _t as reset, a as prop, at as state, gt as next, j as set_text, k as index, lt as pop, rt as set, st as user_derived, ut as push, vt as noop, x as set_style, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { t as Core } from "../chunks/BwrZ3UQO.js";
import "../chunks/DwKmqXYU.js";
import { a as formatTime, l as throttle, n as Loading, r as Notify } from "../chunks/DOFkf9MZ.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
import "../chunks/CGFlWCTf.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as Modal } from "../chunks/iQLQpuQM.js";
import { t as Layer } from "../chunks/C1AypaHL.js";
import { t as SearchSelect } from "../chunks/BOmyLNB5.js";
import { i as postSede, n as PaisCiudadesService, r as postAlmacen, t as AlmacenesService } from "../chunks/BVhuLbLV.js";
var root_1$1 = from_html(`<div class="bg-red-100 text-red-700 p-8 rounded">No hay espacios en al almacén. Agregue uno pulsando en (+)</div>`);
var root_3$1 = from_html(`<th> </th>`);
var root_5$1 = from_html(`<td class="relative py-2 px-2" style="height: 2.6rem"><!></td>`);
var root_4 = from_html(`<tr><td class="text-center"> </td><!></tr>`);
var root_2$1 = from_html(`<div class="_1 bg-white rounded-lg shadow-sm p-8 mb-12 aMz"><div class="w-full flex items-center justify-between px-8 py-8"><div class="flex items-center"><!> <span class="ff-bold text-slate-600">Filas</span> <!> <span class="ff-bold text-slate-600">Niveles</span> <!></div> <button class="bnr4 b-red" aria-label="Eliminar Layout" style="margin-top: -12px; margin-right: -4px"><i class="icon-trash"></i></button></div> <table class="w-full"><thead><tr><th style="width: 3rem">-</th><!></tr></thead><tbody></tbody></table></div>`);
var root = from_html(`<div class="w-full h-full relative"><div class="flex justify-between w-full mb-8 px-12 pt-12"><div></div> <div class="flex items-center"><button class="bx-green" aria-label="Agregar Layout"><i class="icon-plus"></i></button></div></div> <div class="overflow-auto px-4" style="max-height: calc(100% - 60px)"><!> <!></div></div>`);
function AlmacenLayoutEditor($$anchor, $$props) {
	push($$props, true);
	let almacen = prop($$props, "almacen", 15);
	const layouts = user_derived(() => almacen().Layout || []);
	const addLayout = () => {
		const maxID = get(layouts).length > 0 ? Math.max(...get(layouts).map((x) => x.ID || 0)) : 0;
		get(layouts).push({
			RowCant: 2,
			ColCant: 3,
			Name: "",
			ID: maxID + 1,
			Bloques: []
		});
		almacen(almacen().Layout = [...get(layouts)], true);
	};
	const removeLayout = (idx) => {
		almacen(almacen().Layout = get(layouts).filter((_, i) => i !== idx), true);
	};
	const updateLayout = () => {
		almacen(almacen().Layout = [...get(layouts)], true);
	};
	var div = root();
	var div_1 = child(div);
	var div_2 = sibling(child(div_1), 2);
	var button = child(div_2);
	button.__click = addLayout;
	reset(div_2);
	reset(div_1);
	var div_3 = sibling(div_1, 2);
	var node = child(div_3);
	var consequent = ($$anchor$1) => {
		append($$anchor$1, root_1$1());
	};
	if_block(node, ($$render) => {
		if (get(layouts).length === 0) $$render(consequent);
	});
	each(sibling(node, 2), 19, () => get(layouts), (_, idx) => get(layouts)[idx].ID, ($$anchor$1, _, idx) => {
		const layout = user_derived(() => get(layouts)[get(idx)]);
		const heads = user_derived(() => Array.from({ length: get(layout).ColCant || 1 }, (_$1, i) => String(i + 1)));
		const rows = user_derived(() => Array.from({ length: get(layout).RowCant || 1 }, (_$1, i) => String(i + 1)));
		var div_5 = root_2$1();
		var div_6 = child(div_5);
		var div_7 = child(div_6);
		var node_2 = child(div_7);
		Input(node_2, {
			save: "Name",
			css: "shadow-small bg-solid no-border w-220 mr-12",
			inputCss: "text-sm",
			required: true,
			get saveOn() {
				return get(layouts)[get(idx)];
			},
			set saveOn($$value) {
				get(layouts)[get(idx)] = $$value;
			}
		});
		var node_3 = sibling(node_2, 4);
		Input(node_3, {
			save: "RowCant",
			css: "shadow-small bg-solid no-border w-60 mx-4",
			inputCss: "text-sm",
			type: "number",
			onChange: updateLayout,
			get saveOn() {
				return get(layouts)[get(idx)];
			},
			set saveOn($$value) {
				get(layouts)[get(idx)] = $$value;
			}
		});
		Input(sibling(node_3, 4), {
			save: "ColCant",
			css: "shadow-small bg-solid no-border w-60 mx-4",
			inputCss: "text-sm",
			type: "number",
			onChange: updateLayout,
			get saveOn() {
				return get(layouts)[get(idx)];
			},
			set saveOn($$value) {
				get(layouts)[get(idx)] = $$value;
			}
		});
		reset(div_7);
		var button_1 = sibling(div_7, 2);
		button_1.__click = () => removeLayout(get(idx));
		reset(div_6);
		var table = sibling(div_6, 2);
		var thead = child(table);
		var tr = child(thead);
		each(sibling(child(tr)), 17, () => get(heads), index, ($$anchor$2, head) => {
			var th = root_3$1();
			var text = child(th, true);
			reset(th);
			template_effect(() => {
				set_style(th, `width: calc(92% / ${get(heads).length ?? ""})`);
				set_text(text, get(head));
			});
			append($$anchor$2, th);
		});
		reset(tr);
		reset(thead);
		var tbody = sibling(thead);
		each(tbody, 21, () => get(rows), index, ($$anchor$2, row) => {
			var tr_1 = root_4();
			var td = child(tr_1);
			var text_1 = child(td, true);
			reset(td);
			each(sibling(td), 17, () => get(heads), index, ($$anchor$3, col) => {
				var td_1 = root_5$1();
				var node_7 = child(td_1);
				{
					let $0 = user_derived(() => `xy_${get(row)}_${get(col)}`);
					Input(node_7, {
						label: "",
						get save() {
							return get($0);
						},
						css: "shadow-small bg-solid no-border w-full",
						inputCss: "text-sm text-center",
						get saveOn() {
							return get(layouts)[get(idx)];
						},
						set saveOn($$value) {
							get(layouts)[get(idx)] = $$value;
						}
					});
				}
				reset(td_1);
				append($$anchor$3, td_1);
			});
			reset(tr_1);
			template_effect(() => set_text(text_1, get(row)));
			append($$anchor$2, tr_1);
		});
		reset(tbody);
		reset(table);
		reset(div_5);
		append($$anchor$1, div_5);
	});
	reset(div_3);
	reset(div);
	append($$anchor, div);
	pop();
}
delegate(["click"]);
var root_2 = from_html(`<div class="flex items-center justify-between mb-6"><div class="i-search mr-16 w-256"><div><i class="icon-search"></i></div> <input class="w-full" autocomplete="off" type="text"/></div> <div class="flex items-center"><button class="bx-green" aria-label="Agregar Sede"><i class="icon-plus"></i></button></div></div> <!>`, 1);
var root_6 = from_html(`<div class="flex items-center"><div class="ff-bold h3"> </div> <i class="icon-folder-empty"></i> <div class="mr-4 ml-4 h6 text-slate-500">X</div> <div class="ff-bold h3"> </div> <i class="icon-buffer"></i> <div class="mr-4 ml-4 h6 text-slate-500">X</div> <div class="ff-bold h3"> </div> <i class="icon-cube"></i></div>`);
var root_7 = from_html(`<div></div>`);
var root_5 = from_html(`<div class="w-full flex items-center justify-between"><!> <button class="bnr2 b-blue b-card-1" aria-label="Editar Layout"><i class="icon-pencil"></i></button></div>`);
var root_3 = from_html(`<div class="w-full"><div class="flex items-center justify-between mb-6"><div class="i-search mr-16 w-256"><div><i class="icon-search"></i></div> <input class="w-full" autocomplete="off" type="text"/></div> <div class="flex items-center"><button class="bx-green" aria-label="Agregar Almacén"><i class="icon-plus"></i></button></div></div> <!></div>`);
var root_8 = from_html(`<div class="grid grid-cols-24 gap-10"><!> <!> <!> <!> <!></div>`);
var root_9 = from_html(`<div class="grid grid-cols-24 gap-10"><!> <!> <!></div>`);
var root_1 = from_html(`<!> <!> <!> <!> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	const almacenesService = new AlmacenesService();
	const paisCiudadesService = new PaisCiudadesService();
	const pageOptions = [{
		id: 1,
		name: "Sedes"
	}, {
		id: 2,
		name: "Almacenes"
	}];
	let filterText = state("");
	let sedeForm = state(proxy({}));
	let almacenForm = state(proxy({}));
	const saveSede = async (isDelete) => {
		const form = get(sedeForm);
		if ((form.Nombre?.length || 0) < 4 || (form.Direccion?.length || 0) < 4) {
			Notify.failure("El nombre y la dirección deben tener al menos 4 caracteres.");
			return;
		}
		console.log("guardando sede::", form);
		Loading.standard("Creando /Actualizando Sede...");
		try {
			var result = await postSede(form);
		} catch (error) {
			Notify.failure(error);
			Loading.remove();
			return;
		}
		let sedes_ = [...almacenesService.Sedes];
		if (form.ID) {
			const selected = almacenesService.Sedes.find((x) => x.ID === form.ID);
			if (selected) Object.assign(selected, form);
			if (isDelete) sedes_ = sedes_.filter((x) => x.ID !== form.ID);
		} else {
			form.ID = result.ID;
			sedes_.unshift(form);
		}
		almacenesService.Sedes = sedes_;
		Core.closeModal(1);
		Loading.remove();
	};
	const saveAlmacen = async (isDelete) => {
		const form = get(almacenForm);
		if ((form.Nombre?.length || 0) < 4) {
			Notify.failure("El nombre debe tener al menos 4 caracteres.");
			return;
		} else if (!form.SedeID) {
			Notify.failure("Debe seleccionar una sede.");
			return;
		}
		for (let layout of form.Layout || []) {
			layout.Bloques = [];
			for (let key in layout) if (key.substring(0, 3) === "xy_") {
				const [_, rw, co] = key.split("_");
				layout.Bloques.push({
					nm: layout[key],
					rw: parseInt(rw),
					co: parseInt(co)
				});
			}
		}
		console.log("guardando almacen::", form);
		Loading.standard("Creando /Actualizando Almacén...");
		try {
			var result = await postAlmacen(form);
		} catch (error) {
			Notify.failure(error);
			Loading.remove();
			return;
		}
		let almacenes_ = [...almacenesService.Almacenes];
		if (form.ID) {
			const selected = almacenesService.Almacenes.find((x) => x.ID === form.ID);
			if (selected) Object.assign(selected, form);
			if (isDelete) almacenes_ = almacenes_.filter((x) => x.ID !== form.ID);
		} else {
			form.ID = result.ID;
			almacenes_.unshift(form);
		}
		almacenesService.Almacenes = almacenes_;
		Core.closeModal(2);
		Core.hideSideLayer();
		Loading.remove();
	};
	const sedesColumns = [
		{
			header: "ID",
			headerCss: "w-32",
			cellCss: "text-center text-purple-600 px-6",
			getValue: (e) => e.ID
		},
		{
			header: "Nombre",
			cellCss: "px-6",
			getValue: (e) => e.Nombre
		},
		{
			header: "Dirección",
			cellCss: "px-6",
			getValue: (e) => e.Direccion
		},
		{
			header: "Ciudad",
			getValue: (e) => {
				if (!e.Ciudad) return "";
				const arr = e.Ciudad.split("|");
				return arr[1] + " > " + arr[0];
			}
		},
		{
			header: "Actualizado",
			headerCss: "w-144",
			cellCss: "whitespace-nowrap px-6",
			getValue: (e) => formatTime(e.upd, "Y-m-d h:n")
		},
		{
			header: "...",
			headerCss: "w-32",
			cellCss: "text-center px-6",
			buttonEditHandler: (e) => {
				set(sedeForm, { ...e }, true);
				Core.openModal(1);
			}
		}
	];
	const almacenesColumns = [
		{
			header: "ID",
			headerCss: "w-32",
			cellCss: "text-center text-purple-600 px-6",
			getValue: (e) => e.ID
		},
		{
			header: "Sede",
			cellCss: "px-6",
			getValue: (e) => {
				return almacenesService.SedesMap.get(e.SedeID)?.Nombre || `Sede-${e.SedeID}`;
			}
		},
		{
			header: "Nombre",
			cellCss: "px-6",
			getValue: (e) => e.Nombre
		},
		{
			header: "Layout",
			id: "layout",
			cellCss: "px-6",
			getValue: (e) => ""
		},
		{
			header: "Estado",
			getValue: (e) => e.ss
		},
		{
			header: "Actualizado",
			headerCss: "w-144",
			cellCss: "whitespace-nowrap px-6",
			getValue: (e) => formatTime(e.upd, "Y-m-d h:n")
		},
		{
			header: "...",
			headerCss: "w-32",
			cellCss: "text-center px-6",
			buttonEditHandler: (e) => {
				set(almacenForm, JSON.parse(JSON.stringify(e)), true);
				Core.openModal(2);
			}
		}
	];
	const filteredSedes = user_derived(() => {
		if (!get(filterText)) return almacenesService.Sedes;
		const text = get(filterText).toLowerCase();
		return almacenesService.Sedes.filter((e) => {
			return e.Nombre?.toLowerCase().includes(text) || e.Direccion?.toLowerCase().includes(text) || e.Ciudad?.toLowerCase().includes(text);
		});
	});
	const filteredAlmacenes = user_derived(() => {
		if (!get(filterText)) return almacenesService.Almacenes;
		const text = get(filterText).toLowerCase();
		return almacenesService.Almacenes.filter((e) => {
			const sede = almacenesService.SedesMap.get(e.SedeID);
			return e.Nombre?.toLowerCase().includes(text) || sede?.Nombre?.toLowerCase().includes(text);
		});
	});
	const handleLayoutEdit = (almacen) => {
		set(almacenForm, JSON.parse(JSON.stringify(almacen)), true);
		for (let layout of get(almacenForm).Layout || []) for (let e of layout.Bloques || []) layout[`xy_${e.rw}_${e.co}`] = e.nm;
		console.log("ejecutando open side");
		Core.openSideLayer(1);
	};
	Page($$anchor, {
		title: "Sedes & Almacenes",
		get options() {
			return pageOptions;
		},
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var node = first_child(fragment_1);
			var consequent = ($$anchor$2) => {
				var fragment_2 = root_2();
				var div = first_child(fragment_2);
				var div_1 = child(div);
				var input = sibling(child(div_1), 2);
				input.__keyup = (ev) => {
					ev.stopPropagation();
					throttle(() => {
						set(filterText, (ev.target.value || "").toLowerCase().trim(), true);
					}, 150);
				};
				reset(div_1);
				var div_2 = sibling(div_1, 2);
				var button = child(div_2);
				button.__click = (ev) => {
					ev.stopPropagation();
					set(sedeForm, { ss: 1 }, true);
					Core.openModal(1);
				};
				reset(div_2);
				reset(div);
				VTable(sibling(div, 2), {
					css: "w-full",
					maxHeight: "calc(80vh - 13rem)",
					get columns() {
						return sedesColumns;
					},
					get data() {
						return get(filteredSedes);
					}
				});
				append($$anchor$2, fragment_2);
			};
			if_block(node, ($$render) => {
				if (Core.pageOptionSelected === 1) $$render(consequent);
			});
			var node_2 = sibling(node, 2);
			var consequent_3 = ($$anchor$2) => {
				var div_3 = root_3();
				var div_4 = child(div_3);
				var div_5 = child(div_4);
				var input_1 = sibling(child(div_5), 2);
				input_1.__keyup = (ev) => {
					ev.stopPropagation();
					throttle(() => {
						set(filterText, (ev.target.value || "").toLowerCase().trim(), true);
					}, 150);
				};
				reset(div_5);
				var div_6 = sibling(div_5, 2);
				var button_1 = child(div_6);
				button_1.__click = (ev) => {
					ev.stopPropagation();
					set(almacenForm, {
						ID: 0,
						SedeID: 0,
						Nombre: "",
						Descripcion: "",
						ss: 1,
						upd: 0,
						Layout: []
					}, true);
					Core.openModal(2);
				};
				reset(div_6);
				reset(div_4);
				var node_3 = sibling(div_4, 2);
				{
					const cellRenderer = ($$anchor$3, record = noop, col = noop) => {
						var fragment_3 = comment();
						var node_4 = first_child(fragment_3);
						var consequent_2 = ($$anchor$4) => {
							var div_7 = root_5();
							var node_5 = child(div_7);
							var consequent_1 = ($$anchor$5) => {
								const avgCols = user_derived(() => record().Layout.reduce((sum, x) => sum + (x.ColCant || 0), 0) / record().Layout.length);
								const avgRows = user_derived(() => record().Layout.reduce((sum, x) => sum + (x.RowCant || 0), 0) / record().Layout.length);
								var div_8 = root_6();
								var div_9 = child(div_8);
								var text_1 = child(div_9, true);
								reset(div_9);
								var div_10 = sibling(div_9, 6);
								var text_2 = child(div_10, true);
								reset(div_10);
								var div_11 = sibling(div_10, 6);
								var text_3 = child(div_11, true);
								reset(div_11);
								next(2);
								reset(div_8);
								template_effect(($0, $1) => {
									set_text(text_1, record().Layout.length);
									set_text(text_2, $0);
									set_text(text_3, $1);
								}, [() => get(avgCols).toFixed(0), () => get(avgRows).toFixed(0)]);
								append($$anchor$5, div_8);
							};
							var alternate = ($$anchor$5) => {
								append($$anchor$5, root_7());
							};
							if_block(node_5, ($$render) => {
								if (record().Layout && record().Layout.length > 0) $$render(consequent_1);
								else $$render(alternate, false);
							});
							var button_2 = sibling(node_5, 2);
							button_2.__click = () => handleLayoutEdit(record());
							reset(div_7);
							append($$anchor$4, div_7);
						};
						if_block(node_4, ($$render) => {
							if (col().id === "layout") $$render(consequent_2);
						});
						append($$anchor$3, fragment_3);
					};
					VTable(node_3, {
						css: "w-full",
						maxHeight: "calc(80vh - 13rem)",
						get columns() {
							return almacenesColumns;
						},
						get data() {
							return get(filteredAlmacenes);
						},
						cellRenderer,
						$$slots: { cellRenderer: true }
					});
				}
				reset(div_3);
				append($$anchor$2, div_3);
			};
			if_block(node_2, ($$render) => {
				if (Core.pageOptionSelected === 2) $$render(consequent_3);
			});
			var node_6 = sibling(node_2, 2);
			{
				let $0 = user_derived(() => (get(sedeForm)?.ID > 0 ? "Actualizar" : "Crear") + " Sede");
				let $1 = user_derived(() => get(sedeForm)?.ID > 0 ? () => {
					saveSede(true);
				} : void 0);
				Modal(node_6, {
					id: 1,
					get title() {
						return get($0);
					},
					size: 7,
					bodyCss: "px-16 py-14",
					onSave: () => {
						saveSede();
					},
					get onDelete() {
						return get($1);
					},
					children: ($$anchor$2, $$slotProps$1) => {
						var div_13 = root_8();
						var node_7 = child(div_13);
						{
							let $0$1 = user_derived(() => get(sedeForm)?.ID > 0);
							Input(node_7, {
								save: "Nombre",
								css: "col-span-24 md:col-span-10",
								label: "Nombre",
								required: true,
								get disabled() {
									return get($0$1);
								},
								get saveOn() {
									return get(sedeForm);
								},
								set saveOn($$value) {
									set(sedeForm, $$value, true);
								}
							});
						}
						var node_8 = sibling(node_7, 2);
						Input(node_8, {
							save: "Descripcion",
							css: "col-span-24 md:col-span-14",
							label: "Descripción",
							get saveOn() {
								return get(sedeForm);
							},
							set saveOn($$value) {
								set(sedeForm, $$value, true);
							}
						});
						var node_9 = sibling(node_8, 2);
						{
							let $0$1 = user_derived(() => get(sedeForm)?.ID > 0);
							Input(node_9, {
								save: "Telefono",
								css: "col-span-24 md:col-span-10",
								label: "Teléfono",
								get disabled() {
									return get($0$1);
								},
								get saveOn() {
									return get(sedeForm);
								},
								set saveOn($$value) {
									set(sedeForm, $$value, true);
								}
							});
						}
						var node_10 = sibling(node_9, 2);
						Input(node_10, {
							save: "Direccion",
							css: "col-span-24 md:col-span-14",
							label: "Dirección",
							required: true,
							get saveOn() {
								return get(sedeForm);
							},
							set saveOn($$value) {
								set(sedeForm, $$value, true);
							}
						});
						SearchSelect(sibling(node_10, 2), {
							save: "CiudadID",
							css: "col-span-24",
							label: "Departamento | Provincia | Distrito",
							keyId: "ID",
							keyName: "_nombre",
							get options() {
								return paisCiudadesService.distritos;
							},
							required: true,
							get saveOn() {
								return get(sedeForm);
							},
							set saveOn($$value) {
								set(sedeForm, $$value, true);
							}
						});
						reset(div_13);
						append($$anchor$2, div_13);
					},
					$$slots: { default: true }
				});
			}
			var node_12 = sibling(node_6, 2);
			{
				let $0 = user_derived(() => (get(almacenForm)?.ID > 0 ? "Actualizar" : "Crear") + " Almacén");
				let $1 = user_derived(() => get(almacenForm)?.ID > 0 ? () => {
					saveAlmacen(true);
				} : void 0);
				Modal(node_12, {
					id: 2,
					get title() {
						return get($0);
					},
					size: 7,
					bodyCss: "px-16 py-14",
					onSave: () => {
						saveAlmacen();
					},
					get onDelete() {
						return get($1);
					},
					children: ($$anchor$2, $$slotProps$1) => {
						var div_14 = root_9();
						var node_13 = child(div_14);
						SearchSelect(node_13, {
							save: "SedeID",
							css: "col-span-24 md:col-span-12",
							label: "Sede",
							keyId: "ID",
							keyName: "Nombre",
							get options() {
								return almacenesService.Sedes;
							},
							required: true,
							get saveOn() {
								return get(almacenForm);
							},
							set saveOn($$value) {
								set(almacenForm, $$value, true);
							}
						});
						var node_14 = sibling(node_13, 2);
						Input(node_14, {
							save: "Nombre",
							css: "col-span-24 md:col-span-12",
							label: "Nombre",
							required: true,
							get saveOn() {
								return get(almacenForm);
							},
							set saveOn($$value) {
								set(almacenForm, $$value, true);
							}
						});
						Input(sibling(node_14, 2), {
							save: "Descripcion",
							css: "col-span-24",
							label: "Descripción",
							get saveOn() {
								return get(almacenForm);
							},
							set saveOn($$value) {
								set(almacenForm, $$value, true);
							}
						});
						reset(div_14);
						append($$anchor$2, div_14);
					},
					$$slots: { default: true }
				});
			}
			var node_16 = sibling(node_12, 2);
			{
				let $0 = user_derived(() => "Layout " + (get(almacenForm)?.Nombre || "-"));
				Layer(node_16, {
					id: 1,
					type: "side",
					get title() {
						return get($0);
					},
					contentCss: "p-0",
					css: "px-8 py-8 md:px-14 md:py-10",
					titleCss: "h2 ff-bold",
					onClose: () => {},
					onSave: () => {
						saveAlmacen();
					},
					children: ($$anchor$2, $$slotProps$1) => {
						AlmacenLayoutEditor($$anchor$2, {
							get almacen() {
								return get(almacenForm);
							},
							set almacen($$value) {
								set(almacenForm, $$value, true);
							}
						});
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
