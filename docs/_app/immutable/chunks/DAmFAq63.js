import { $ as proxy, at as state, rt as set, z as get } from "./CTB4HzdN.js";
import { h as POST, m as GetHandler } from "./BwrZ3UQO.js";
var ProductosService = class extends GetHandler {
	route = "productos";
	useCache = {
		min: 5,
		ver: 9
	};
	#productos = state(proxy([]));
	get productos() {
		return get(this.#productos);
	}
	set productos(value) {
		set(this.#productos, value, true);
	}
	#productosMap = state(proxy(/* @__PURE__ */ new Map()));
	get productosMap() {
		return get(this.#productosMap);
	}
	set productosMap(value) {
		set(this.#productosMap, value, true);
	}
	handler(result) {
		console.log("productos result::", result);
		for (const e of result) {
			e.Image = e.Images?.[0];
			e.CategoriasIDs = e.CategoriasIDs || [];
		}
		this.productos = result.filter((x) => x.ss);
		this.productosMap = new Map(result.map((x) => [x.ID, x]));
	}
	constructor() {
		super();
		this.fetch();
	}
};
const postProducto = (data) => {
	return POST({
		data,
		route: "productos",
		refreshRoutes: ["productos"]
	});
};
var ListasCompartidasService = class extends GetHandler {
	route = "listas-compartidas2";
	useCache = {
		min: 5,
		ver: 6
	};
	#Records = state(proxy([]));
	get Records() {
		return get(this.#Records);
	}
	set Records(value) {
		set(this.#Records, value, true);
	}
	#RecordsMap = state(proxy(/* @__PURE__ */ new Map()));
	get RecordsMap() {
		return get(this.#RecordsMap);
	}
	set RecordsMap(value) {
		set(this.#RecordsMap, value, true);
	}
	#ListaRecordsMap = state(proxy(/* @__PURE__ */ new Map()));
	get ListaRecordsMap() {
		return get(this.#ListaRecordsMap);
	}
	set ListaRecordsMap(value) {
		set(this.#ListaRecordsMap, value, true);
	}
	get(id) {
		return this.RecordsMap.get(id);
	}
	handler(result) {
		console.log("result getted::", result);
		this.Records = [];
		this.ListaRecordsMap = /* @__PURE__ */ new Map();
		for (const [key, records] of Object.entries(result)) {
			const listaID = parseInt(key.split("_")[1]);
			for (const e of records) {
				if (!e.ss) continue;
				this.Records.push(e);
				if ([1, 2].includes(listaID)) {
					const imagesMap = new Map((e.Images || []).filter((x) => x).map((x) => [parseInt(x.split("-")[1]), x]));
					e.Images = [];
					for (const order of [
						1,
						2,
						3
					]) e.Images.push(imagesMap.get(order) || "");
				}
				this.ListaRecordsMap.has(e.ListaID) ? this.ListaRecordsMap.get(e.ListaID)?.push(e) : this.ListaRecordsMap.set(e.ListaID, [e]);
			}
		}
		this.RecordsMap = new Map(this.Records.map((x) => [x.ID, x]));
	}
	addNew(e) {
		const records = this.ListaRecordsMap.get(e.ListaID) || [];
		records.unshift(e);
		this.ListaRecordsMap.set(e.ListaID, [...records]);
		this.ListaRecordsMap = new Map(this.ListaRecordsMap);
		this.RecordsMap.set(e.ID, e);
		this.Records.push(e);
	}
	constructor(ids = []) {
		super();
		if (ids.length > 0) this.route = `listas-compartidas2?ids=${ids.join(",")}`;
		if (ids) this.fetch();
	}
};
const postListaRegistros = (data) => {
	return POST({
		data,
		route: "listas-compartidas",
		refreshRoutes: ["listas-compartidas2"]
	});
};
const productoAtributos = [
	{
		id: 1,
		name: "Color"
	},
	{
		id: 2,
		name: "Talla"
	},
	{
		id: 3,
		name: "Tamaño"
	},
	{
		id: 4,
		name: "Forma"
	},
	{
		id: 5,
		name: "Presentación"
	}
];
export { productoAtributos as a, postProducto as i, ProductosService as n, postListaRegistros as r, ListasCompartidasService as t };
