import { $ as proxy, at as state, rt as set, z as get } from "./CTB4HzdN.js";
import { h as POST, m as GetHandler, p as GET } from "./BwrZ3UQO.js";
import { r as Notify } from "./DOFkf9MZ.js";
var CajasService = class extends GetHandler {
	route = "cajas";
	useCache = {
		min: 1,
		ver: 1
	};
	#Cajas = state(proxy([]));
	get Cajas() {
		return get(this.#Cajas);
	}
	set Cajas(value) {
		set(this.#Cajas, value, true);
	}
	#CajasMap = state(proxy(/* @__PURE__ */ new Map()));
	get CajasMap() {
		return get(this.#CajasMap);
	}
	set CajasMap(value) {
		set(this.#CajasMap, value, true);
	}
	handler(result) {
		console.log("result cajas::", result);
		this.Cajas = result.Cajas || [];
		for (let e of this.Cajas) e.SaldoCurrent = e.SaldoCurrent || 0;
		this.CajasMap = new Map(this.Cajas.map((x) => [x.ID, x]));
	}
	constructor() {
		super();
		this.fetch();
	}
};
const postCaja = (data) => {
	return POST({
		data,
		route: "cajas",
		refreshRoutes: ["cajas"]
	});
};
var arrayToMapN = (arr, key) => {
	return new Map(arr.map((x) => [x[key], x]));
};
const getCajaMovimientos = async (args) => {
	let route = `caja-movimientos?caja-id=${args.CajaID}`;
	if ((!args.fechaInicio || !args.fechaFin) && !args.lastRegistros) throw "No se encontró una fecha de inicio o fin.";
	if (args.fechaInicio && args.fechaFin) {
		route += `&fecha-hora-inicio=${args.fechaInicio}`;
		route += `&fecha-hora-fin=${args.fechaFin + 1}`;
	}
	if (args.lastRegistros) route += `&last-registros=${args.lastRegistros}`;
	let result;
	try {
		result = await GET({ route });
	} catch (error) {
		console.log("Error:", error);
		Notify.failure(error);
		throw error;
	}
	const usuariosMap = arrayToMapN(result.usuarios, "id");
	for (let e of result.movimientos) e.Usuario = usuariosMap.get(e.CreatedBy);
	return result.movimientos;
};
const postCajaMovimiento = (data) => {
	return POST({
		data,
		route: "caja-movimiento",
		refreshRoutes: ["cajas"]
	});
};
const postCajaCuadre = (data) => {
	return POST({
		data,
		route: "caja-cuadre",
		refreshRoutes: ["cajas"]
	});
};
const getCajaCuadres = async (args) => {
	let route = `caja-cuadres?caja-id=${args.CajaID}`;
	if ((!args.fechaInicio || !args.fechaFin) && !args.lastRegistros) throw "No se encontró una fecha de inicio o fin.";
	if (args.fechaInicio && args.fechaFin) {
		route += `&fecha-hora-inicio=${args.fechaInicio * 24 * 60 * 60}`;
		route += `&fecha-hora-fin=${(args.fechaFin + 1) * 24 * 60 * 60}`;
	}
	if (args.lastRegistros) route += `&last-registros=${args.lastRegistros}`;
	let result;
	try {
		result = await GET({ route });
	} catch (error) {
		console.log("Error:", error);
		Notify.failure(error);
		throw error;
	}
	console.log("Result Cajas::", result);
	const usuariosMap = arrayToMapN(result.usuarios, "id");
	for (const e of result.cuadres || []) e.Usuario = usuariosMap.get(e.CreatedBy);
	return result.cuadres;
};
const cajaTipos = [{
	id: 1,
	name: "Caja"
}, {
	id: 2,
	name: "Cuenta Bancaria"
}];
const cajaMovimientoTipos = [
	{
		id: 1,
		name: "-",
		group: 1
	},
	{
		id: 2,
		name: "Cuadre Físico",
		group: 1
	},
	{
		id: 3,
		name: "Transferencia",
		group: 2,
		isNegative: true
	},
	{
		id: 4,
		name: "Retiro",
		group: 2,
		isNegative: true
	},
	{
		id: 5,
		name: "Pérdida",
		group: 2,
		isNegative: true
	},
	{
		id: 6,
		name: "Pago",
		group: 2,
		isNegative: true
	},
	{
		id: 7,
		name: "Cobro",
		group: 2
	}
];
export { getCajaMovimientos as a, postCajaMovimiento as c, getCajaCuadres as i, cajaMovimientoTipos as n, postCaja as o, cajaTipos as r, postCajaCuadre as s, CajasService as t };
