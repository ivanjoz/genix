import { $ as proxy, G as user_effect, L as delegate, M as append, P as from_html, Q as sibling, V as untrack, X as child, Z as first_child, _t as reset, at as state, lt as pop, rt as set, st as user_derived, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { p as GET } from "../chunks/BwrZ3UQO.js";
import "../chunks/DwKmqXYU.js";
import { a as formatTime, l as throttle, n as Loading, o as highlString, r as Notify } from "../chunks/DOFkf9MZ.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as SearchSelect } from "../chunks/BOmyLNB5.js";
import { t as DateInput } from "../chunks/ArQKZ6r_.js";
import { t as AlmacenesService } from "../chunks/BVhuLbLV.js";
import { n as ProductosService } from "../chunks/DAmFAq63.js";
const movimientoTipos = [{
	id: 1,
	name: "Entrada Manual"
}, {
	id: 2,
	name: "Salida Manual"
}];
const queryAlmacenMovimientos = async (args) => {
	let route = `almacen-movimientos?almacen-id=${args.almacenID}`;
	if (!args.fechaInicio || !args.fechaFin) throw "No se encontró una fecha de inicio o fin.";
	const zoneOffset = -18e3;
	route += `&fecha-hora-inicio=${args.fechaInicio * 24 * 60 * 60 + zoneOffset}`;
	route += `&fecha-hora-fin=${(args.fechaFin + 1) * 24 * 60 * 60 + zoneOffset}`;
	let result;
	try {
		result = await GET({
			route,
			errorMessage: "Hubo un error al obtener los movimientos del almacén"
		});
	} catch (error) {
		Notify.failure(error);
		throw error;
	}
	console.log("almacén movimientos:", result);
	result.Usuarios = result.Usuarios || [];
	result.Productos = result.Productos || [];
	result.Movimientos = result.Movimientos || [];
	result.Movimientos.sort((a, b) => b.Created - a.Created);
	return result;
};
var root_1 = from_html(`<div class="flex items-center justify-between mb-12"><div class="flex items-center w-full" style="max-width: 64rem;"><!> <!> <!> <button class="px-16 py-8 bx-purple mt-8 h-44" aria-label="Consultar registros"><i class="icon-search"></i></button></div> <div class="flex items-center mr-16 w-224 ml-auto relative"><div class="absolute left-12 text-gray-400"><i class="icon-search"></i></div> <input class="w-full pl-36 pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500" autocomplete="off" type="text" placeholder="Buscar..."/></div></div> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	const almacenes = new AlmacenesService();
	new ProductosService();
	const getFechaUnix = () => {
		return Math.floor(Date.now() / (1e3 * 60 * 60 * 24));
	};
	const fechaFin = getFechaUnix();
	let form = state(proxy({
		fechaFin,
		fechaInicio: fechaFin - 7,
		almacenID: 0
	}));
	let almacenMovimientos = state(proxy([]));
	let filterText = state("");
	const usuariosMap = /* @__PURE__ */ new Map();
	const productosMap = /* @__PURE__ */ new Map();
	const movimientoTiposMap = new Map(movimientoTipos.map((x) => [x.id, x]));
	user_effect(() => {
		if (almacenes?.Almacenes?.length > 0) untrack(() => {
			const almacenID = (almacenes.Almacenes || [])[0]?.ID;
			set(form, {
				...get(form),
				almacenID
			}, true);
		});
	});
	const consultarRegistros = async () => {
		if (!get(form).almacenID) return;
		Loading.standard("Consultando registros...");
		let result;
		try {
			result = await queryAlmacenMovimientos(get(form));
		} catch (error) {
			Loading.remove();
			return;
		}
		console.log("registros obtenidos: ", result);
		for (const e of result.Productos) productosMap.set(e.ID, e);
		for (const e of result.Usuarios) usuariosMap.set(e.id, e);
		set(almacenMovimientos, result.Movimientos, true);
		Loading.remove();
	};
	const almacenRender = (almacenID, cant) => {
		if (!almacenID) return "";
		return `<div class="flex items-center">
      <div class="mr-8">${almacenes.AlmacenesMap.get(almacenID)?.Nombre || `Almacen-${almacenID}`}</div>
      <div class="ff-mono text-blue-600">(</div>
      <div class="ff-mono">${cant}</div>
      <div class="ff-mono text-blue-600">)</div>
    </div>`;
	};
	const columns = [
		{
			header: "Fecha Hora",
			headerCss: "w-120",
			cellCss: "ff-mono px-6",
			getValue: (e) => formatTime(e.Created, "d-M h:n")
		},
		{
			header: "Producto",
			render: (e) => {
				const segments = highlString(productosMap.get(e.ProductoID)?.Nombre || `Producto-${e.ProductoID}`, get(filterText).toLowerCase().trim().split(" ").filter((x) => x));
				let html = "";
				for (const seg of segments) if (seg.highl) html += `<span class="bg-yellow-200">${seg.text}</span>`;
				else html += seg.text;
				return html;
			}
		},
		{
			header: "Lote",
			headerCss: "w-100",
			cellCss: "text-purple-600 text-center px-6",
			getValue: (e) => e.Lote || "-"
		},
		{
			header: "SKU",
			headerCss: "w-100",
			cellCss: "text-purple-600 text-center px-6",
			getValue: (e) => e.SKU || "-"
		},
		{
			header: "Movimiento",
			headerCss: "w-120",
			cellCss: "text-center px-6",
			render: (e) => {
				return movimientoTiposMap.get(e.Tipo)?.name || "-";
			}
		},
		{
			header: "Cantidad",
			headerCss: "w-100",
			cellCss: "text-right ff-mono px-6",
			render: (e) => {
				return `<div class="flex justify-end ${e.Cantidad < 0 ? "text-red-500" : "text-blue-600"}">
          ${e.Cantidad}
        </div>`;
			}
		},
		{
			header: "Almacén Origen",
			render: (e) => almacenRender(e.AlmacenOrigenID, e.AlmacenOrigenCantidad)
		},
		{
			header: "Almacén Destino",
			render: (e) => almacenRender(e.AlmacenID, e.AlmacenCantidad)
		},
		{
			header: "Usuario",
			headerCss: "w-120",
			cellCss: "text-center px-6",
			render: (e) => {
				const segments = highlString(usuariosMap.get(e.CreatedBy || 1)?.usuario || `Usuario-${e.CreatedBy}`, get(filterText).toLowerCase().trim().split(" ").filter((x) => x));
				let html = "";
				for (const seg of segments) if (seg.highl) html += `<span class="bg-yellow-200">${seg.text}</span>`;
				else html += seg.text;
				return html;
			}
		}
	];
	Page($$anchor, {
		title: "Almacén Movimientos",
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = first_child(fragment_1);
			var div_1 = child(div);
			var node = child(div_1);
			{
				let $0 = user_derived(() => almacenes?.Almacenes || []);
				SearchSelect(node, {
					save: "almacenID",
					css: "w-240 mr-12",
					label: "Almacén",
					keyId: "ID",
					keyName: "Nombre",
					get options() {
						return get($0);
					},
					placeholder: "",
					required: true,
					get saveOn() {
						return get(form);
					},
					set saveOn($$value) {
						set(form, $$value, true);
					}
				});
			}
			var node_1 = sibling(node, 2);
			DateInput(node_1, {
				label: "Fecha Inicio",
				css: "w-140 mr-12",
				save: "fechaInicio",
				get saveOn() {
					return get(form);
				},
				set saveOn($$value) {
					set(form, $$value, true);
				}
			});
			var node_2 = sibling(node_1, 2);
			DateInput(node_2, {
				label: "Fecha Fin",
				css: "w-140 mr-12",
				save: "fechaFin",
				get saveOn() {
					return get(form);
				},
				set saveOn($$value) {
					set(form, $$value, true);
				}
			});
			var button = sibling(node_2, 2);
			button.__click = (ev) => {
				ev.stopPropagation();
				consultarRegistros();
			};
			reset(div_1);
			var div_2 = sibling(div_1, 2);
			var input = sibling(child(div_2), 2);
			input.__keyup = (ev) => {
				ev.stopPropagation();
				throttle(() => {
					set(filterText, (ev.target.value || "").toLowerCase().trim(), true);
				}, 150);
			};
			reset(div_2);
			reset(div);
			VTable(sibling(div, 2), {
				get data() {
					return get(almacenMovimientos);
				},
				get columns() {
					return columns;
				},
				css: "w-full",
				tableCss: "w-full",
				maxHeight: "calc(100vh - 8rem - 12px)",
				get filterText() {
					return get(filterText);
				},
				getFilterContent: (e) => {
					return [productosMap.get(e.ProductoID)?.Nombre || "", usuariosMap.get(e.CreatedBy || 1)?.usuario || ""].join(" ").toLowerCase();
				},
				useFilterCache: true
			});
			append($$anchor$1, fragment_1);
		},
		$$slots: { default: true }
	});
	pop();
}
delegate(["click", "keyup"]);
export { _page as component };
