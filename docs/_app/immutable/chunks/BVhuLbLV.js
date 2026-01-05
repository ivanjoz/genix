import { $ as proxy, at as state, rt as set, z as get } from "./CTB4HzdN.js";
import { h as POST, m as GetHandler } from "./BwrZ3UQO.js";
var AlmacenesService = class extends GetHandler {
	route = "sedes-almacenes";
	useCache = {
		min: 5,
		ver: 1
	};
	#Almacenes = state(proxy([]));
	get Almacenes() {
		return get(this.#Almacenes);
	}
	set Almacenes(value) {
		set(this.#Almacenes, value, true);
	}
	#AlmacenesMap = state(proxy(/* @__PURE__ */ new Map()));
	get AlmacenesMap() {
		return get(this.#AlmacenesMap);
	}
	set AlmacenesMap(value) {
		set(this.#AlmacenesMap, value, true);
	}
	#Sedes = state(proxy([]));
	get Sedes() {
		return get(this.#Sedes);
	}
	set Sedes(value) {
		set(this.#Sedes, value, true);
	}
	#SedesMap = state(proxy(/* @__PURE__ */ new Map()));
	get SedesMap() {
		return get(this.#SedesMap);
	}
	set SedesMap(value) {
		set(this.#SedesMap, value, true);
	}
	handler(result) {
		console.log("sedes almacenes::", result);
		this.Almacenes = result.Almacenes || [];
		this.Sedes = result.Sedes || [];
		this.SedesMap = new Map(this.Sedes.map((e) => [e.ID, e]));
		this.AlmacenesMap = new Map(this.Almacenes.map((e) => [e.ID, e]));
	}
	constructor() {
		super();
		this.fetch();
	}
};
const postSede = (data) => {
	return POST({
		data,
		route: "sedes",
		refreshRoutes: ["sedes-almacenes"]
	});
};
const postAlmacen = (data) => {
	return POST({
		data,
		route: "almacenes",
		refreshRoutes: ["sedes-almacenes"]
	});
};
var PaisCiudadesService = class extends GetHandler {
	route = "pais-ciudades?pais-id=604";
	useCache = {
		min: 600,
		ver: 1
	};
	#ciudades = state(proxy([]));
	get ciudades() {
		return get(this.#ciudades);
	}
	set ciudades(value) {
		set(this.#ciudades, value, true);
	}
	#distritos = state(proxy([]));
	get distritos() {
		return get(this.#distritos);
	}
	set distritos(value) {
		set(this.#distritos, value, true);
	}
	#ciudadesMap = state(proxy(/* @__PURE__ */ new Map()));
	get ciudadesMap() {
		return get(this.#ciudadesMap);
	}
	set ciudadesMap(value) {
		set(this.#ciudadesMap, value, true);
	}
	handler(result) {
		const ciudades = result?.filter((x) => !x._IS_META) || [];
		const distritos = [];
		const ciudadesMap = new Map(ciudades.map((e) => [e.ID, e]));
		for (let e of ciudades) {
			const padre = ciudadesMap.get(e.PadreID);
			if (e.Jerarquia === 3) distritos.push(e);
			if (padre) {
				if (padre.Jerarquia === 2) e.Provincia = padre;
				else if (padre.Jerarquia === 1) e.Departamento = padre;
				if (padre.PadreID && ciudadesMap.has(padre.PadreID)) {
					const padre2 = ciudadesMap.get(padre.PadreID);
					if (padre2?.Jerarquia === 2) e.Provincia = padre2;
					else if (padre2?.Jerarquia === 1) e.Departamento = padre2;
				}
			}
		}
		for (let e of distritos) e._nombre = `${e.Departamento?.Nombre || "-"} ► ${e.Provincia?.Nombre || ""} ► ${e.Nombre}`;
		console.log("distritos:", distritos);
		this.ciudades = ciudades;
		this.distritos = distritos;
		this.ciudadesMap = ciudadesMap;
	}
	constructor() {
		super();
		this.fetch();
	}
};
export { postSede as i, PaisCiudadesService as n, postAlmacen as r, AlmacenesService as t };
