import { $ as proxy, A as if_block, G as user_effect, L as delegate, M as append, N as comment, P as from_html, Q as sibling, S as set_class, W as template_effect, X as child, Z as first_child, _t as reset, at as state, j as set_text, lt as pop, rt as set, st as user_derived, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { b as formatN, t as Core } from "../chunks/BwrZ3UQO.js";
import "../chunks/DwKmqXYU.js";
import { a as formatTime, l as throttle, n as Loading, r as Notify } from "../chunks/DOFkf9MZ.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
import { t as OptionsStrip } from "../chunks/CGFlWCTf.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as Modal } from "../chunks/iQLQpuQM.js";
import { t as SearchSelect } from "../chunks/BOmyLNB5.js";
import { t as AlmacenesService } from "../chunks/BVhuLbLV.js";
import { t as Checkbox } from "../chunks/w0aS_awe.js";
import { a as getCajaMovimientos, c as postCajaMovimiento, i as getCajaCuadres, n as cajaMovimientoTipos, o as postCaja, r as cajaTipos, s as postCajaCuadre, t as CajasService } from "../chunks/DBgcaFpj.js";
var root_3 = from_html(`<div class="flex justify-center items-center py-40"><div class="text-slate-400">Cargando...</div></div>`);
var root_5 = from_html(`<div class="bg-red-100 text-red-700 p-8 mt-8 rounded">Seleccione una Caja</div>`);
var root_6 = from_html(`<div class="flex w-full justify-between mt-8"><div class="flex items-center"><div class="text-[1.1rem] ff-bold mr-8"> </div> <button class="bx-yellow" aria-label="edit"><i class="icon-pencil"></i></button></div> <div class="flex items-center"><button class="bx-green" aria-label="add"><i class="icon-plus"></i></button></div></div> <!>`, 1);
var root_8 = from_html(`<div class="flex justify-center items-center py-40"><div class="text-slate-400">Cargando...</div></div>`);
var root_10 = from_html(`<div class="bg-red-100 text-red-700 p-8 mt-8 rounded">Seleccione una Caja</div>`);
var root_11 = from_html(`<div class="flex w-full justify-between mt-8"><div></div> <div class="flex items-center"><button class="bx-green" aria-label="add"><i class="icon-plus"></i></button></div></div> <!>`, 1);
var root_12 = from_html(`<div class="grid grid-cols-24 gap-10"><!> <!> <!> <!> <div class="col-span-24 flex justify-between items-center"><div></div> <!></div></div>`);
var root_15 = from_html(`<span> </span>`);
var root_16 = from_html(`<div class="col-span-24 text-red-600 ff-bold"><i class="icon-attention"></i> </div>`);
var root_13 = from_html(`<div class="flex items-start w-full p-16"><div class="w-260 mr-16"><div class="w-full mb-12"><div class="text-sm mb-4 text-slate-600">Saldo Sistema</div> <div class="text-[1.1rem] ff-mono text-center bg-slate-100 py-8 rounded"> </div></div> <!> <div class="mb-12"></div> <div class="w-full"><div class="text-sm mb-4 text-slate-600">Diferencia</div> <div class="text-[1.1rem] min-h-32 ff-mono text-center bg-slate-100 py-8 rounded"><!></div></div></div> <div class="flex items-end"><button class="bx-purple w-full mt-24"><i class="text-[0.875rem] icon-arrows-cw"></i> Recalcular</button></div> <!></div>`);
var root_18 = from_html(`<span> </span>`);
var root_17 = from_html(`<div class="grid grid-cols-24 gap-10"><!> <!> <!> <div class="col-span-24 md:col-span-12"></div> <div class="col-span-24 md:col-span-12"><div class="text-sm mb-4 text-slate-600">Saldo Inicial</div> <div class="text-[1.1rem] ff-mono text-center bg-slate-100 py-8 rounded"> </div></div> <div class="col-span-24 md:col-span-12"><div class="text-sm mb-4 text-slate-600">Saldo Final</div> <div class="text-[1.1rem] ff-mono text-center bg-slate-100 py-8 rounded"><!></div></div></div>`);
var root_1 = from_html(`<div class="flex flex-col md:flex-row justify-between mb-6 gap-10"><div class="w-full md:w-[36%]"><div class="flex justify-between items-center w-full mb-10"><div class="i-search mr-16 w-256"><div><i class="icon-search"></i></div> <input class="w-full" autocomplete="off" type="text"/></div> <div class="flex items-center"><button class="bx-green" aria-label="agregar"><i class="icon-plus"></i></button></div></div> <!></div> <div class="w-full md:w-[calc(64%-22px)] bg-white rounded-md shadow-sm p-12"><!> <!> <!></div></div> <!> <!> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	const cajaMovimientoTiposMap = new Map(cajaMovimientoTipos.map((x) => [x.id, x]));
	const almacenes = new AlmacenesService();
	const cajas = new CajasService();
	let filterText = state("");
	let layerView = state(1);
	let cajaForm = state(proxy({}));
	let cajaCuadreForm = state(proxy({}));
	let cajaMovimientoForm = state(proxy({}));
	let cajaMovimientos = state(proxy([]));
	let cajaCuadres = state(proxy([]));
	let isLoadingMovimientos = state(false);
	let isLoadingCuadres = state(false);
	const columns = [
		{
			header: "ID",
			headerCss: "w-32",
			cellCss: "text-center text-purple-600 px-6",
			getValue: (e) => e.ID
		},
		{
			header: "Nombre",
			cellCss: "px-6 py-2",
			getValue: (e) => e.Nombre,
			render: (e) => {
				const tipoName = cajaTipos.find((x) => x.id === e.Tipo)?.name || "-";
				return `<div class="leading-tight">
          <div class="ff-bold leading-[1.1] h3">${e.Nombre}</div>
          <div class="fs15 text-slate-500">${tipoName}</div>
        </div>`;
			}
		},
		{
			header: "Cuadre",
			getValue: (e) => e.CuadreFecha ? String(e.CuadreFecha) : "",
			render: (e) => {
				if (!e.CuadreFecha) return "";
				return `<div class="leading-tight text-right">
          <div class="ff-mono text-[0.875rem]">${formatN(e.CuadreSaldo / 100, 2)}</div>
          <div class="text-[0.875rem] text-slate-500">${formatTime(e.CuadreFecha, "d-M h:n")}</div>
        </div>`;
			}
		},
		{
			header: "Saldo",
			cellCss: "text-right ff-mono px-6",
			getValue: (e) => formatN(e.SaldoCurrent / 100, 2)
		}
	];
	const saveCaja = async () => {
		const caja = get(cajaForm);
		if (!caja.Nombre || !caja.Tipo || !caja.SedeID) {
			Notify.failure("Los inputs Nombre, Tipo y Sede son obligatorios");
			return;
		}
		Loading.standard("Guardando caja...");
		try {
			var result = await postCaja(caja);
		} catch (error) {
			console.warn(error);
			return;
		}
		Loading.remove();
		caja.SaldoCurrent = caja.SaldoCurrent || 0;
		const selected = cajas.CajasMap.get(caja.ID);
		if (selected) Object.assign(selected, caja);
		else {
			caja.ID = result.ID;
			cajas.Cajas.push(caja);
		}
		cajas.Cajas = [...cajas.Cajas];
		Core.closeModal(1);
	};
	const saveCajaCuadre = async () => {
		const form = get(cajaCuadreForm);
		form.SaldoSistema = get(cajaForm).SaldoCurrent;
		Loading.standard("Guardando caja...");
		let recordSaved;
		try {
			recordSaved = await postCajaCuadre(form);
		} catch (error) {
			console.warn(error);
			return;
		}
		Loading.remove();
		const caja = cajas.CajasMap.get(form.CajaID);
		if (!caja) return;
		if (typeof recordSaved?.NeedUpdateSaldo === "number") {
			caja.SaldoCurrent = recordSaved.NeedUpdateSaldo;
			set(cajaForm, { ...caja }, true);
			const newForm = { ...get(cajaCuadreForm) };
			newForm._error = `Hubo una actualización en el saldo de la caja. El saldo actual es "${formatN(caja.SaldoCurrent / 100, 2)}". Intente nuevamente con el cálculo actualizado.`;
			newForm.SaldoDiferencia = newForm.SaldoReal - caja.SaldoCurrent;
			set(cajaCuadreForm, newForm, true);
		} else {
			caja.SaldoCurrent = form.SaldoReal;
			cajas.Cajas = [...cajas.Cajas];
			Object.assign(get(cajaForm), caja);
			Core.closeModal(2);
			get(cajaCuadres).unshift(recordSaved);
		}
	};
	const saveCajaMovimiento = async () => {
		const form = get(cajaMovimientoForm);
		if (!form.Tipo || !form.Monto) {
			Notify.failure("Se necesita seleccionar un monto y un tipo.");
			return;
		}
		Loading.standard("Guardando Movimiento...");
		let movimientoSaved;
		try {
			movimientoSaved = await postCajaMovimiento(form);
		} catch (error) {
			console.warn(error);
			return;
		}
		Loading.remove();
		const caja = cajas.CajasMap.get(form.CajaID);
		if (!caja) return;
		caja.SaldoCurrent = form.SaldoFinal;
		cajas.Cajas = [...cajas.Cajas];
		Object.assign(get(cajaForm), caja);
		get(cajaMovimientos).unshift(movimientoSaved);
		Core.closeModal(3);
	};
	const isCajaMovimiento = user_derived(() => [3].includes(get(cajaMovimientoForm).Tipo));
	user_effect(() => {
		if (get(layerView) === 1 && get(cajaForm).ID) {
			set(isLoadingMovimientos, true);
			getCajaMovimientos({
				CajaID: get(cajaForm).ID,
				lastRegistros: 200
			}).then((result) => {
				set(cajaMovimientos, result, true);
			}).catch((error) => {
				Notify.failure(error);
			}).finally(() => {
				set(isLoadingMovimientos, false);
			});
		}
	});
	user_effect(() => {
		if (get(layerView) === 2 && get(cajaForm).ID) {
			set(isLoadingCuadres, true);
			getCajaCuadres({
				CajaID: get(cajaForm).ID,
				lastRegistros: 200
			}).then((result) => {
				set(cajaCuadres, result, true);
			}).catch((error) => {
				console.log("Error:", error);
				Notify.failure(error);
			}).finally(() => {
				set(isLoadingCuadres, false);
			});
		}
	});
	const filteredCajas = user_derived(() => {
		if (!get(filterText)) return cajas.Cajas;
		const text = get(filterText).toLowerCase();
		return cajas.Cajas.filter((e) => {
			return e.Nombre?.toLowerCase().includes(text);
		});
	});
	Page($$anchor, {
		title: "Cajas & Bancos",
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
				set(cajaForm, {
					ID: -1,
					ss: 1
				}, true);
				Core.openModal(1);
			};
			reset(div_4);
			reset(div_2);
			VTable(sibling(div_2, 2), {
				css: "w-full",
				get columns() {
					return columns;
				},
				maxHeight: "calc(100vh - 8rem - 16px)",
				get data() {
					return get(filteredCajas);
				},
				get selected() {
					return get(cajaForm).ID;
				},
				isSelected: (e, id) => e?.ID === id,
				tableCss: "cursor-pointer",
				onRowClick: (record) => {
					set(cajaForm, get(cajaForm).ID === record.ID ? {} : { ...record }, true);
				}
			});
			reset(div_1);
			var div_5 = sibling(div_1, 2);
			var node_1 = child(div_5);
			OptionsStrip(node_1, {
				get selected() {
					return get(layerView);
				},
				options: [[1, "Movimientos"], [2, "Cuadres"]],
				buttonCss: "ff-bold",
				onSelect: (e) => {
					set(layerView, e[0], true);
				}
			});
			var node_2 = sibling(node_1, 2);
			var consequent_2 = ($$anchor$2) => {
				var fragment_2 = comment();
				var node_3 = first_child(fragment_2);
				var consequent = ($$anchor$3) => {
					append($$anchor$3, root_3());
				};
				var alternate_1 = ($$anchor$3) => {
					var fragment_3 = comment();
					var node_4 = first_child(fragment_3);
					var consequent_1 = ($$anchor$4) => {
						append($$anchor$4, root_5());
					};
					var alternate = ($$anchor$4) => {
						var fragment_4 = root_6();
						var div_8 = first_child(fragment_4);
						var div_9 = child(div_8);
						var div_10 = child(div_9);
						var text_1 = child(div_10, true);
						reset(div_10);
						var button_1 = sibling(div_10, 2);
						button_1.__click = (ev) => {
							ev.stopPropagation();
							Core.openModal(1);
						};
						reset(div_9);
						var div_11 = sibling(div_9, 2);
						var button_2 = child(div_11);
						button_2.__click = (ev) => {
							ev.stopPropagation();
							Core.openModal(3);
							set(cajaMovimientoForm, {
								CajaID: get(cajaForm).ID,
								SaldoFinal: get(cajaForm).SaldoCurrent
							}, true);
						};
						reset(div_11);
						reset(div_8);
						VTable(sibling(div_8, 2), {
							css: "w-full mt-8",
							maxHeight: "calc(100vh - 8rem - 16px)",
							get data() {
								return get(cajaMovimientos);
							},
							columns: [
								{
									header: "Fecha Hora",
									getValue: (e) => formatTime(e.Created, "d-M h:n")
								},
								{
									header: "Tipo Mov.",
									getValue: (e) => cajaMovimientoTiposMap.get(e.Tipo)?.name || ""
								},
								{
									header: "Monto",
									cellCss: "ff-mono text-right px-6",
									render: (e) => {
										const monto = formatN(e.Monto / 100, 2);
										return `<span class="${e.Monto < 0 ? "text-red-600" : ""}">${monto}</span>`;
									}
								},
								{
									header: "Saldo Final",
									cellCss: "ff-mono text-right px-6",
									getValue: (e) => formatN(e.SaldoFinal / 100, 2)
								},
								{
									header: "Nº Documento",
									getValue: (e) => ""
								},
								{
									header: "Usuario",
									cellCss: "text-center px-6",
									getValue: (e) => e.Usuario?.usuario || ""
								}
							]
						});
						template_effect(() => set_text(text_1, get(cajaForm)?.Nombre || ""));
						append($$anchor$4, fragment_4);
					};
					if_block(node_4, ($$render) => {
						if (!get(cajaForm).ID) $$render(consequent_1);
						else $$render(alternate, false);
					}, true);
					append($$anchor$3, fragment_3);
				};
				if_block(node_3, ($$render) => {
					if (get(isLoadingMovimientos)) $$render(consequent);
					else $$render(alternate_1, false);
				});
				append($$anchor$2, fragment_2);
			};
			if_block(node_2, ($$render) => {
				if (get(layerView) === 1) $$render(consequent_2);
			});
			var node_6 = sibling(node_2, 2);
			var consequent_5 = ($$anchor$2) => {
				var fragment_5 = comment();
				var node_7 = first_child(fragment_5);
				var consequent_3 = ($$anchor$3) => {
					append($$anchor$3, root_8());
				};
				var alternate_3 = ($$anchor$3) => {
					var fragment_6 = comment();
					var node_8 = first_child(fragment_6);
					var consequent_4 = ($$anchor$4) => {
						append($$anchor$4, root_10());
					};
					var alternate_2 = ($$anchor$4) => {
						var fragment_7 = root_11();
						var div_14 = first_child(fragment_7);
						var div_15 = sibling(child(div_14), 2);
						var button_3 = child(div_15);
						button_3.__click = (ev) => {
							ev.stopPropagation();
							Core.openModal(2);
							set(cajaCuadreForm, { CajaID: get(cajaForm).ID }, true);
						};
						reset(div_15);
						reset(div_14);
						VTable(sibling(div_14, 2), {
							css: "w-full mt-8",
							maxHeight: "calc(100vh - 8rem - 16px)",
							get data() {
								return get(cajaCuadres);
							},
							columns: [
								{
									header: "Fecha Hora",
									getValue: (e) => formatTime(e.Created, "d-M h:n")
								},
								{
									header: "Saldo Sistema",
									cellCss: "ff-mono text-right px-6",
									getValue: (e) => formatN((e.SaldoSistema || 0) / 100, 2)
								},
								{
									header: "Diferencia",
									cellCss: "ff-mono text-right px-6",
									getValue: (e) => formatN((e.SaldoDiferencia || 0) / 100, 2)
								},
								{
									header: "Saldo Real",
									cellCss: "ff-mono text-right px-6",
									getValue: (e) => formatN((e.SaldoReal || 0) / 100, 2)
								},
								{
									header: "Usuario",
									cellCss: "text-center px-6",
									getValue: (e) => e.Usuario?.usuario || ""
								}
							]
						});
						append($$anchor$4, fragment_7);
					};
					if_block(node_8, ($$render) => {
						if (!get(cajaForm).ID) $$render(consequent_4);
						else $$render(alternate_2, false);
					}, true);
					append($$anchor$3, fragment_6);
				};
				if_block(node_7, ($$render) => {
					if (get(isLoadingCuadres)) $$render(consequent_3);
					else $$render(alternate_3, false);
				});
				append($$anchor$2, fragment_5);
			};
			if_block(node_6, ($$render) => {
				if (get(layerView) === 2) $$render(consequent_5);
			});
			reset(div_5);
			reset(div);
			var node_10 = sibling(div, 2);
			Modal(node_10, {
				id: 1,
				title: "Cajas",
				size: 6,
				bodyCss: "px-16 py-14",
				onSave: () => {
					saveCaja();
				},
				onDelete: () => {},
				children: ($$anchor$2, $$slotProps$1) => {
					var div_16 = root_12();
					var node_11 = child(div_16);
					SearchSelect(node_11, {
						save: "Tipo",
						css: "col-span-24 md:col-span-10",
						label: "Tipo",
						keyId: "id",
						keyName: "name",
						get options() {
							return cajaTipos;
						},
						placeholder: "",
						required: true,
						get saveOn() {
							return get(cajaForm);
						},
						set saveOn($$value) {
							set(cajaForm, $$value, true);
						}
					});
					var node_12 = sibling(node_11, 2);
					Input(node_12, {
						save: "Nombre",
						css: "col-span-24 md:col-span-14",
						label: "Nombre",
						required: true,
						get saveOn() {
							return get(cajaForm);
						},
						set saveOn($$value) {
							set(cajaForm, $$value, true);
						}
					});
					var node_13 = sibling(node_12, 2);
					Input(node_13, {
						save: "Descripcion",
						css: "col-span-24",
						label: "Descripcion",
						get saveOn() {
							return get(cajaForm);
						},
						set saveOn($$value) {
							set(cajaForm, $$value, true);
						}
					});
					var node_14 = sibling(node_13, 2);
					SearchSelect(node_14, {
						save: "SedeID",
						css: "col-span-24 md:col-span-10",
						label: "Sede",
						required: true,
						get options() {
							return almacenes.Sedes;
						},
						keyId: "ID",
						keyName: "Nombre",
						get saveOn() {
							return get(cajaForm);
						},
						set saveOn($$value) {
							set(cajaForm, $$value, true);
						}
					});
					var div_17 = sibling(node_14, 2);
					Checkbox(sibling(child(div_17), 2), {
						label: "Saldo Negativo",
						save: "Nombre",
						get saveOn() {
							return get(cajaForm);
						},
						set saveOn($$value) {
							set(cajaForm, $$value, true);
						}
					});
					reset(div_17);
					reset(div_16);
					append($$anchor$2, div_16);
				},
				$$slots: { default: true }
			});
			var node_16 = sibling(node_10, 2);
			Modal(node_16, {
				id: 2,
				title: "Cuadre de Caja",
				size: 6,
				onSave: () => {
					saveCajaCuadre();
				},
				children: ($$anchor$2, $$slotProps$1) => {
					var div_18 = root_13();
					var div_19 = child(div_18);
					var div_20 = child(div_19);
					var div_21 = sibling(child(div_20), 2);
					var text_2 = child(div_21, true);
					reset(div_21);
					reset(div_20);
					var node_17 = sibling(div_20, 2);
					Input(node_17, {
						save: "SaldoReal",
						type: "number",
						inputCss: "text-[1.1rem] ff-mono text-center",
						baseDecimals: 2,
						css: "w-full mb-12",
						label: "Saldo Encontrado",
						required: true,
						onChange: () => {
							console.log("caja cuadre::", get(cajaCuadreForm));
							set(cajaCuadreForm, { ...get(cajaCuadreForm) }, true);
						},
						get saveOn() {
							return get(cajaCuadreForm);
						},
						set saveOn($$value) {
							set(cajaCuadreForm, $$value, true);
						}
					});
					var div_22 = sibling(node_17, 4);
					var div_23 = sibling(child(div_22), 2);
					var node_18 = child(div_23);
					var consequent_7 = ($$anchor$3) => {
						const diff = user_derived(() => (get(cajaCuadreForm).SaldoReal || 0) - get(cajaForm).SaldoCurrent);
						var fragment_8 = comment();
						var node_19 = first_child(fragment_8);
						var consequent_6 = ($$anchor$4) => {
							var span = root_15();
							var text_3 = child(span, true);
							reset(span);
							template_effect(($0) => {
								set_class(span, 1, get(diff) > 0 ? "text-blue-600" : "text-red-600");
								set_text(text_3, $0);
							}, [() => formatN(get(diff) / 100, 2)]);
							append($$anchor$4, span);
						};
						if_block(node_19, ($$render) => {
							if (get(diff)) $$render(consequent_6);
						});
						append($$anchor$3, fragment_8);
					};
					if_block(node_18, ($$render) => {
						if (get(cajaCuadreForm).SaldoReal) $$render(consequent_7);
					});
					reset(div_23);
					reset(div_22);
					reset(div_19);
					var node_20 = sibling(div_19, 4);
					var consequent_8 = ($$anchor$3) => {
						var div_24 = root_16();
						var text_4 = sibling(child(div_24), 1, true);
						reset(div_24);
						template_effect(() => set_text(text_4, get(cajaCuadreForm)._error));
						append($$anchor$3, div_24);
					};
					if_block(node_20, ($$render) => {
						if (get(cajaCuadreForm)._error) $$render(consequent_8);
					});
					reset(div_18);
					template_effect(($0) => set_text(text_2, $0), [() => formatN((get(cajaForm).SaldoCurrent || 0) / 100, 2)]);
					append($$anchor$2, div_18);
				},
				$$slots: { default: true }
			});
			Modal(sibling(node_16, 2), {
				id: 3,
				title: "Movimiento de Caja",
				size: 6,
				onSave: () => {
					saveCajaMovimiento();
				},
				children: ($$anchor$2, $$slotProps$1) => {
					var div_25 = root_17();
					var node_22 = child(div_25);
					{
						let $0 = user_derived(() => cajaMovimientoTipos.filter((x) => x.group === 2));
						SearchSelect(node_22, {
							save: "Tipo",
							css: "col-span-24 md:col-span-12",
							label: "Tipo",
							keyId: "id",
							keyName: "name",
							get options() {
								return get($0);
							},
							placeholder: "",
							required: true,
							onChange: () => {
								get(cajaMovimientoForm).CajaRefID = 0;
								set(cajaMovimientoForm, { ...get(cajaMovimientoForm) }, true);
							},
							get saveOn() {
								return get(cajaMovimientoForm);
							},
							set saveOn($$value) {
								set(cajaMovimientoForm, $$value, true);
							}
						});
					}
					var node_23 = sibling(node_22, 2);
					{
						let $0 = user_derived(() => !get(isCajaMovimiento));
						let $1 = user_derived(() => get(isCajaMovimiento) ? "seleccione" : "no aplica");
						SearchSelect(node_23, {
							save: "CajaRefID",
							css: "col-span-24 md:col-span-12",
							label: "Caja Destino",
							keyId: "id",
							keyName: "name",
							get options() {
								return cajaTipos;
							},
							get disabled() {
								return get($0);
							},
							get placeholder() {
								return get($1);
							},
							required: true,
							get saveOn() {
								return get(cajaMovimientoForm);
							},
							set saveOn($$value) {
								set(cajaMovimientoForm, $$value, true);
							}
						});
					}
					var node_24 = sibling(node_23, 2);
					Input(node_24, {
						save: "Monto",
						inputCss: "ff-mono text-[1.1rem] text-center",
						css: "col-span-24 md:col-span-12",
						label: "Monto",
						baseDecimals: 2,
						required: true,
						type: "number",
						transform: (v) => {
							const movTipo = cajaMovimientoTiposMap.get(get(cajaMovimientoForm).Tipo);
							console.log("movimiento tipo::", movTipo);
							if (movTipo?.isNegative && typeof v === "number" && v > 0) v = v * -1;
							return v;
						},
						onChange: () => {
							const form = { ...get(cajaMovimientoForm) };
							form.SaldoFinal = get(cajaForm).SaldoCurrent + (form.Monto || 0);
							set(cajaMovimientoForm, form, true);
						},
						get saveOn() {
							return get(cajaMovimientoForm);
						},
						set saveOn($$value) {
							set(cajaMovimientoForm, $$value, true);
						}
					});
					var div_26 = sibling(node_24, 4);
					var div_27 = sibling(child(div_26), 2);
					var text_5 = child(div_27, true);
					reset(div_27);
					reset(div_26);
					var div_28 = sibling(div_26, 2);
					var div_29 = sibling(child(div_28), 2);
					var node_25 = child(div_29);
					var consequent_9 = ($$anchor$3) => {
						const saldo = user_derived(() => get(cajaMovimientoForm).SaldoFinal);
						var span_1 = root_18();
						var text_6 = child(span_1, true);
						reset(span_1);
						template_effect(($0) => {
							set_class(span_1, 1, get(saldo) >= 0 ? "" : "text-red-600");
							set_text(text_6, $0);
						}, [() => formatN(get(saldo) / 100, 2)]);
						append($$anchor$3, span_1);
					};
					if_block(node_25, ($$render) => {
						if (get(cajaMovimientoForm).SaldoFinal !== void 0) $$render(consequent_9);
					});
					reset(div_29);
					reset(div_28);
					reset(div_25);
					template_effect(($0) => set_text(text_5, $0), [() => formatN(get(cajaForm).SaldoCurrent / 100, 2)]);
					append($$anchor$2, div_25);
				},
				$$slots: { default: true }
			});
			append($$anchor$1, fragment_1);
		},
		$$slots: { default: true }
	});
	pop();
}
delegate(["keyup", "click"]);
export { _page as component };
