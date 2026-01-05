import { $ as proxy, G as user_effect, L as delegate, M as append, P as from_html, Q as sibling, V as untrack, X as child, Z as first_child, _t as reset, at as state, lt as pop, rt as set, st as user_derived, ut as push, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { b as formatN } from "../chunks/BwrZ3UQO.js";
import "../chunks/DwKmqXYU.js";
import { a as formatTime, l as throttle, n as Loading, r as Notify } from "../chunks/DOFkf9MZ.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as SearchSelect } from "../chunks/BOmyLNB5.js";
import { t as DateInput } from "../chunks/ArQKZ6r_.js";
import { a as getCajaMovimientos, n as cajaMovimientoTipos, t as CajasService } from "../chunks/DBgcaFpj.js";
var root_1 = from_html(`<div class="flex items-center justify-between mb-12"><div class="flex items-center w-full" style="max-width: 64rem;"><!> <!> <!> <button class="px-16 py-8 bx-blue mt-8 h-44" aria-label="Consultar registros"><i class="icon-search"></i></button></div> <div class="flex items-center mr-16 w-224 ml-auto relative"><div class="absolute left-12 text-gray-400"><i class="icon-search"></i></div> <input class="w-full pl-36 pr-12 py-8 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-indigo-500" autocomplete="off" type="text" placeholder="Buscar..."/></div></div> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	const cajas = new CajasService();
	const cajaMovimientoTiposMap = new Map(cajaMovimientoTipos.map((x) => [x.id, x]));
	const getFechaUnix = () => {
		return Math.floor(Date.now() / (1e3 * 60 * 60 * 24));
	};
	const fechaFin = getFechaUnix();
	let form = state(proxy({
		fechaFin,
		fechaInicio: fechaFin - 7,
		CajaID: 0
	}));
	let cajaMovimientos = state(proxy([]));
	let filterText = state("");
	user_effect(() => {
		if (cajas?.Cajas?.length > 0) untrack(() => {
			const CajaID = (cajas.Cajas || [])[0]?.ID;
			set(form, {
				...get(form),
				CajaID
			}, true);
		});
	});
	const consultarRegistros = async () => {
		if (!get(form).CajaID || !get(form).fechaInicio || !get(form).fechaFin) {
			Notify.failure("Debe seleccionar una caja y un rango de fechas.");
			return;
		}
		Loading.standard("Consultando registros...");
		let result;
		try {
			const zoneOffset = -18e3;
			const fechaHoraInicio = get(form).fechaInicio * 24 * 60 * 60 + zoneOffset;
			const fechaHoraFin = (get(form).fechaFin + 1) * 24 * 60 * 60 + zoneOffset;
			result = await getCajaMovimientos({
				CajaID: get(form).CajaID,
				fechaInicio: fechaHoraInicio,
				fechaFin: fechaHoraFin
			});
		} catch (error) {
			Loading.remove();
			return;
		}
		Loading.remove();
		set(cajaMovimientos, result || [], true);
		console.log("movimientos obtenidos: ", result);
	};
	const columns = [
		{
			header: "Fecha Hora",
			headerCss: "w-140",
			cellCss: "ff-mono px-6",
			getValue: (e) => formatTime(e.Created, "d-M h:n")
		},
		{
			header: "Tipo Mov.",
			headerCss: "w-160",
			cellCss: "px-6",
			getValue: (e) => cajaMovimientoTiposMap.get(e.Tipo)?.name || ""
		},
		{
			header: "Monto",
			headerCss: "w-120",
			cellCss: "ff-mono text-right px-6",
			render: (e) => {
				return `<span class="${e.Monto < 0 ? "text-red-500" : ""}">${formatN(e.Monto / 100, 2)}</span>`;
			}
		},
		{
			header: "Saldo Final",
			headerCss: "w-120",
			cellCss: "ff-mono text-right px-6",
			getValue: (e) => formatN(e.SaldoFinal / 100, 2)
		},
		{
			header: "Nº Documento",
			headerCss: "w-140",
			cellCss: "text-center px-6",
			getValue: (e) => ""
		},
		{
			header: "Usuario",
			headerCss: "w-120",
			cellCss: "text-center px-6",
			getValue: (e) => e.Usuario?.usuario || ""
		}
	];
	Page($$anchor, {
		title: "Cajas Movimientos",
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = first_child(fragment_1);
			var div_1 = child(div);
			var node = child(div_1);
			{
				let $0 = user_derived(() => cajas?.Cajas || []);
				SearchSelect(node, {
					save: "CajaID",
					css: "w-240 mr-12",
					label: "Cajas & Bancos",
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
					return get(cajaMovimientos);
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
					return [cajaMovimientoTiposMap.get(e.Tipo)?.name || "", e.Usuario?.usuario || ""].join(" ").toLowerCase();
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
