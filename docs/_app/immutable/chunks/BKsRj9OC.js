import { $ as proxy, at as state, rt as set, z as get } from "./CTB4HzdN.js";
import { h as POST, m as GetHandler } from "./BwrZ3UQO.js";
var UsuariosService = class extends GetHandler {
	route = "usuarios";
	useCache = {
		min: 5,
		ver: 1
	};
	#usuarios = state(proxy([]));
	get usuarios() {
		return get(this.#usuarios);
	}
	set usuarios(value) {
		set(this.#usuarios, value, true);
	}
	#usuariosMap = state(proxy(/* @__PURE__ */ new Map()));
	get usuariosMap() {
		return get(this.#usuariosMap);
	}
	set usuariosMap(value) {
		set(this.#usuariosMap, value, true);
	}
	handler(response) {
		this.usuarios = response || [];
		this.usuariosMap = new Map(this.usuarios.map((x) => [x.id, x]));
	}
	constructor() {
		super();
		this.fetch();
	}
	updateUsuario(usuario) {
		const existing = this.usuarios.find((x) => x.id === usuario.id);
		if (existing) Object.assign(existing, usuario);
		else this.usuarios.unshift(usuario);
		this.usuariosMap.set(usuario.id, usuario);
	}
	removeUsuario(id) {
		this.usuarios = this.usuarios.filter((x) => x.id !== id);
		this.usuariosMap.delete(id);
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
	}
	constructor() {
		super();
		this.fetch();
	}
};
const postUsuario = (data) => {
	return POST({
		data,
		route: "usuarios",
		refreshRoutes: ["usuarios"]
	});
};
const postUsuarioPropio = (data) => {
	return POST({
		data,
		route: "usuario-propio",
		refreshRoutes: ["usuarios"]
	});
};
export { postUsuarioPropio as i, UsuariosService as n, postUsuario as r, PerfilesService as t };
