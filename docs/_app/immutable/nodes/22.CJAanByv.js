import { $ as proxy, A as if_block, G as user_effect, L as delegate, M as append, P as from_html, Q as sibling, Tt as __toESM, V as untrack, X as child, Z as first_child, _t as reset, at as state, lt as pop, mt as snapshot, rt as set, st as user_derived, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { h as POST, p as GET, t as Core } from "../chunks/BwrZ3UQO.js";
import { o as require_notiflix_aio_3_2_8_min } from "../chunks/DwKmqXYU.js";
import { l as throttle, r as Notify } from "../chunks/DOFkf9MZ.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
import "../chunks/CGFlWCTf.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as Layer } from "../chunks/C1AypaHL.js";
import { t as SearchSelect } from "../chunks/BOmyLNB5.js";
import { t as AlmacenesService } from "../chunks/BVhuLbLV.js";
import { n as ProductosService } from "../chunks/DAmFAq63.js";
import { t as Checkbox } from "../chunks/w0aS_awe.js";
var import_notiflix_aio_3_2_8_min = /* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min());
const getProductosStock = async (almacenID) => {
	let records = [];
	try {
		records = await GET({
			route: `productos-stock?almacen-id=${almacenID}`,
			errorMessage: "Hubo un error al obtener el stock.",
			useCache: {
				min: 0,
				ver: 6
			}
		});
	} catch (error) {
		Notify.failure(error);
	}
	return records;
};
const postProductosStock = (data) => {
	return POST({
		data,
		route: "productos-stock",
		refreshRoutes: ["productos-stock"]
	});
};
var root_2 = from_html(`<div class="ml-12 c-red"><i class="icon-attention"></i>Debe seleccionar un almacén.</div>`);
var root_3 = from_html(`<div class="i-search w-full md:w-200 md:ml-12 col-span-5"><div><i class="icon-search"></i></div> <input type="text"/></div> <!>`, 1);
var root_4 = from_html(`<button class="bx-blue mr-8"><i class="icon-floppy"></i>Guardar</button> <button class="bx-green" aria-label="agregar"><i class="icon-plus"></i></button>`, 1);
var root_5 = from_html(`<div class="grid grid-cols-24 gap-10 mt-6 p-4"><!> <!> <!> <!> <!> <!></div>`);
var root_1 = from_html(`<div class="flex items-center mb-8"><!> <!> <div class="ml-auto"><!></div></div> <!> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	const almacenes = new AlmacenesService();
	const productos = new ProductosService();
	let filters = state(proxy({
		almacenID: 0,
		showTodosProductos: false
	}));
	let filterText = state("");
	let almacenStock = state(proxy([]));
	let almacenStockGetted = [];
	let form = state(proxy({}));
	let formProducto = user_derived(() => productos?.productosMap?.get(get(form).ProductoID || 0));
	user_effect(() => {
		if (!get(filters).almacenID) return;
	});
	let columns = [
		{
			header: "Producto",
			highlight: true,
			getValue: (e) => {
				return productos.productosMap.get(e.ProductoID)?.Nombre || "";
			}
		},
		{
			header: "Lote",
			getValue: (e) => e.Lote || ""
		},
		{
			header: "SKU",
			getValue: (e) => e.SKU || ""
		},
		{
			header: "Presentación",
			getValue: (e) => {
				if (!e.PresentacionID) return "";
				return (productos.productosMap.get(e.ProductoID)?.Presentaciones?.find((x) => x.id === e.PresentacionID))?.nm || "";
			}
		},
		{
			header: "Stock",
			css: "justify-end",
			inputCss: "text-right pr-6",
			getValue: (e) => e.Cantidad,
			onCellEdit: (e, value) => {
				agregarStock(e, parseInt(value || "0"));
			},
			render: (e) => {
				if (e._cantidadPrev && e._cantidadPrev !== e.Cantidad) return {
					css: "flex items-center",
					children: [
						{ text: String(e._cantidadPrev > 0 ? e._cantidadPrev : 0) },
						{
							text: "→",
							css: "ml-2 mr-2"
						},
						{
							text: e.Cantidad,
							css: "c-red"
						}
					]
				};
				else return { text: e.Cantidad || "" };
			}
		},
		{
			header: "Costo Un.",
			onCellEdit: (e, value) => {
				e._hasUpdated = true;
				e.CostoUn = parseInt(value || "0");
			}
		}
	];
	const onChangeAlmacen = async () => {
		if (!get(filters).almacenID) return;
		import_notiflix_aio_3_2_8_min.Loading.standard();
		try {
			var result = await getProductosStock(get(filters).almacenID);
		} catch (error) {
			import_notiflix_aio_3_2_8_min.Loading.remove();
			return;
		}
		import_notiflix_aio_3_2_8_min.Loading.remove();
		set(almacenStock, result || [], true);
		almacenStockGetted = result || [];
	};
	const guardarRegistros = async () => {
		const recordsForUpdate = get(almacenStock).filter((e) => e._hasUpdated);
		if (recordsForUpdate.length === 0) {
			Notify.failure("No hay registros a actualizar.");
			return;
		}
		import_notiflix_aio_3_2_8_min.Loading.standard("Enviando registros...");
		let result;
		try {
			result = await postProductosStock(recordsForUpdate);
		} catch (error) {
			import_notiflix_aio_3_2_8_min.Loading.remove();
			return;
		}
		for (const e of get(almacenStock)) e._cantidadPrev = 0;
		console.log("resultado obtenido::", result);
		import_notiflix_aio_3_2_8_min.Loading.remove();
	};
	const fillAllProductos = () => {
		console.log("almacenStockGetted", snapshot(almacenStockGetted));
		const productosStockMap = new Map(almacenStockGetted?.map((x) => [x.ID, x]));
		for (const pr of productos.productos) {
			const presentacionesIDs = pr.Presentaciones?.length > 0 ? pr.Presentaciones.map((x) => x.id) : [0];
			for (const presentacionID of presentacionesIDs) {
				const stockID = [
					get(filters).almacenID,
					pr.ID,
					presentacionID || "",
					"",
					""
				].join("_");
				if (productosStockMap.has(stockID)) {
					const e = productosStockMap.get(stockID);
					e.PresentacionID = presentacionID;
					continue;
				}
				const stock = {
					ID: stockID,
					AlmacenID: get(filters).almacenID,
					ProductoID: pr.ID,
					PresentacionID: presentacionID,
					Cantidad: 0
				};
				productosStockMap.set(stockID, stock);
			}
		}
		set(almacenStock, [...productosStockMap.values()], true);
	};
	const agregarStock = (e, cantidad) => {
		if (!e.ID) e.ID = [
			e.AlmacenID,
			e.ProductoID,
			e.PresentacionID || 0,
			e.SKU || "",
			e.Lote || ""
		].join("_");
		const base = get(almacenStock).find((x) => x.ID === e.ID) || almacenStockGetted.find((x) => x.ID === e.ID);
		if (base) {
			base._hasUpdated = true;
			base._cantidadPrev = base._cantidadPrev || base.Cantidad || -1;
			base.Cantidad = cantidad;
		} else {
			e._cantidadPrev = -1;
			e._hasUpdated = true;
			almacenStockGetted.unshift(e);
			get(almacenStock).unshift(e);
			set(almacenStock, [...get(almacenStock)], true);
		}
	};
	user_effect(() => {
		if (get(filters).showTodosProductos) untrack(() => {
			fillAllProductos();
		});
		else untrack(() => {
			set(almacenStock, almacenStockGetted, true);
		});
	});
	Page($$anchor, {
		sideLayerSize: 640,
		title: "productos-stock",
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = first_child(fragment_1);
			var node = child(div);
			{
				let $0 = user_derived(() => almacenes?.Almacenes || []);
				SearchSelect(node, {
					get options() {
						return get($0);
					},
					keyId: "ID",
					keyName: "Nombre",
					save: "almacenID",
					placeholder: "ALMACÉN ::",
					css: "w-270",
					onChange: () => {
						onChangeAlmacen();
					},
					get saveOn() {
						return get(filters);
					},
					set saveOn($$value) {
						set(filters, $$value, true);
					}
				});
			}
			var node_1 = sibling(node, 2);
			var consequent = ($$anchor$2) => {
				append($$anchor$2, root_2());
			};
			var alternate = ($$anchor$2) => {
				var fragment_2 = root_3();
				var div_2 = first_child(fragment_2);
				var input = sibling(child(div_2), 2);
				input.__keyup = (ev) => {
					const value = String(ev.target.value || "");
					throttle(() => {
						set(filterText, value, true);
					}, 150);
				};
				reset(div_2);
				Checkbox(sibling(div_2, 2), {
					label: "Todos los Productos",
					save: "showTodosProductos",
					css: "ml-16",
					get saveOn() {
						return get(filters);
					},
					set saveOn($$value) {
						set(filters, $$value, true);
					}
				});
				append($$anchor$2, fragment_2);
			};
			if_block(node_1, ($$render) => {
				if (!get(filters).almacenID) $$render(consequent);
				else $$render(alternate, false);
			});
			var div_3 = sibling(node_1, 2);
			var node_3 = child(div_3);
			var consequent_1 = ($$anchor$2) => {
				var fragment_3 = root_4();
				var button = first_child(fragment_3);
				button.__click = () => {
					guardarRegistros();
				};
				var button_1 = sibling(button, 2);
				button_1.__click = () => {
					set(form, { AlmacenID: get(filters).almacenID }, true);
					Core.openSideLayer(1);
				};
				append($$anchor$2, fragment_3);
			};
			if_block(node_3, ($$render) => {
				if (get(filters).almacenID > 0) $$render(consequent_1);
			});
			reset(div_3);
			reset(div);
			var node_4 = sibling(div, 2);
			VTable(node_4, {
				get columns() {
					return columns;
				},
				get data() {
					return get(almacenStock);
				},
				get filterText() {
					return get(filterText);
				},
				useFilterCache: true,
				getFilterContent: (e) => {
					return [
						productos.productosMap.get(e.ProductoID)?.Nombre,
						e.SKU,
						e.Lote
					].filter((x) => x).join(" ").toLowerCase();
				}
			});
			Layer(sibling(node_4, 2), {
				id: 1,
				type: "bottom",
				css: "p-12 min-h-360",
				title: "Agregar Stock",
				titleCss: "h2",
				saveButtonName: "Agregar",
				saveButtonIcon: "icon-ok",
				contentOverflow: true,
				onSave: () => {
					if (!get(form).ProductoID || !get(form).Cantidad) Notify.failure("Debe seleccionar un producto y una cantidad.");
					agregarStock(get(form), get(form).Cantidad);
					Core.hideSideLayer();
				},
				children: ($$anchor$2, $$slotProps$1) => {
					var div_4 = root_5();
					var node_6 = child(div_4);
					{
						let $0 = user_derived(() => productos.productos || []);
						SearchSelect(node_6, {
							label: "Producto",
							css: "col-span-24",
							required: true,
							save: "ProductoID",
							get options() {
								return get($0);
							},
							keyName: "Nombre",
							keyId: "ID",
							get saveOn() {
								return get(form);
							},
							set saveOn($$value) {
								set(form, $$value, true);
							}
						});
					}
					var node_7 = sibling(node_6, 2);
					var consequent_2 = ($$anchor$3) => {
						{
							let $0 = user_derived(() => get(formProducto)?.Presentaciones || []);
							SearchSelect($$anchor$3, {
								label: "Presentación",
								css: "col-span-24",
								save: "PresentacionID",
								get options() {
									return get($0);
								},
								keyName: "nm",
								keyId: "id",
								get saveOn() {
									return get(form);
								},
								set saveOn($$value) {
									set(form, $$value, true);
								}
							});
						}
					};
					if_block(node_7, ($$render) => {
						if ((get(formProducto)?.Presentaciones?.filter((x) => x.ss) || []).length > 0) $$render(consequent_2);
					});
					var node_8 = sibling(node_7, 2);
					Input(node_8, {
						label: "SKU",
						css: "col-span-16",
						save: "SKU",
						get saveOn() {
							return get(form);
						},
						set saveOn($$value) {
							set(form, $$value, true);
						}
					});
					var node_9 = sibling(node_8, 2);
					Input(node_9, {
						label: "Cantidad",
						required: true,
						css: "col-span-8",
						save: "Cantidad",
						type: "number",
						get saveOn() {
							return get(form);
						},
						set saveOn($$value) {
							set(form, $$value, true);
						}
					});
					var node_10 = sibling(node_9, 2);
					Input(node_10, {
						label: "Lote",
						css: "col-span-16",
						save: "Lote",
						get saveOn() {
							return get(form);
						},
						set saveOn($$value) {
							set(form, $$value, true);
						}
					});
					Input(sibling(node_10, 2), {
						label: "Costo x Unidad",
						css: "col-span-8",
						save: "CostoUn",
						type: "number",
						get saveOn() {
							return get(form);
						},
						set saveOn($$value) {
							set(form, $$value, true);
						}
					});
					reset(div_4);
					append($$anchor$2, div_4);
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
