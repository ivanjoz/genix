const __vite__mapDeps=(i,m=__vite__mapDeps,d=(m.f||(m.f=["../nodes/0.Iy9pUNgx.js","./B4yhAxNU.js","./hMJtJoiB.js","../assets/shared.C_sT_agh.css","../nodes/1.DqYHtEL4.js","../nodes/2.7sBeelal.js"])))=>i.map(i=>d[i]);
import { n as __exportAll } from "./hMJtJoiB.js";
import { A as set_text, B as user_pre_effect, C as component, D as each, E as html, F as delegate, G as set, H as first_child, I as get, J as pop, K as state, L as tick, M as comment, N as from_html, O as index, P as text, Q as reset, R as template_effect, S as head, T as slot, U as sibling, V as child, W as proxy, X as setContext, Y as push, Z as next, _ as prop, b as set_class, c as Core, d as browser, g as asClassComponent, h as onMount, i as Input, j as append, k as if_block, l as mainMenuOptions, n as Header, o as getProductos, p as __vitePreload, q as user_derived, r as MainCarrusel, s as productosServiceState, t as ProductCards, u as suscribeUrlFlag, v as bind_this, w as snippet, x as clsx, y as set_attribute, z as user_effect } from "./B4yhAxNU.js";
const matchers = {};
var root_4 = from_html(`<div id="svelte-announcer" aria-live="assertive" aria-atomic="true" style="position: absolute; left: 0; top: 0; clip: rect(0 0 0 0); clip-path: inset(50%); overflow: hidden; white-space: nowrap; width: 1px; height: 1px"><!></div>`);
var root$4 = from_html(`<!> <!>`, 1);
function Root($$anchor, $$props) {
	push($$props, true);
	let components = prop($$props, "components", 23, () => []), data_0 = prop($$props, "data_0", 3, null), data_1 = prop($$props, "data_1", 3, null);
	if (!browser) setContext("__svelte__", $$props.stores);
	if (browser) user_pre_effect(() => $$props.stores.page.set($$props.page));
	else $$props.stores.page.set($$props.page);
	user_effect(() => {
		$$props.stores;
		$$props.page;
		$$props.constructors;
		components();
		$$props.form;
		data_0();
		data_1();
		$$props.stores.page.notify();
	});
	let mounted = state(false);
	let navigated = state(false);
	let title = state(null);
	onMount(() => {
		const unsubscribe = $$props.stores.page.subscribe(() => {
			if (get(mounted)) {
				set(navigated, true);
				tick().then(() => {
					set(title, document.title || "untitled page", true);
				});
			}
		});
		set(mounted, true);
		return unsubscribe;
	});
	const Pyramid_1 = user_derived(() => $$props.constructors[1]);
	var fragment = root$4();
	var node = first_child(fragment);
	var consequent = ($$anchor) => {
		const Pyramid_0 = user_derived(() => $$props.constructors[0]);
		var fragment_1 = comment();
		component(first_child(fragment_1), () => get(Pyramid_0), ($$anchor, Pyramid_0_1) => {
			bind_this(Pyramid_0_1($$anchor, {
				get data() {
					return data_0();
				},
				get form() {
					return $$props.form;
				},
				get params() {
					return $$props.page.params;
				},
				children: ($$anchor, $$slotProps) => {
					var fragment_2 = comment();
					component(first_child(fragment_2), () => get(Pyramid_1), ($$anchor, Pyramid_1_1) => {
						bind_this(Pyramid_1_1($$anchor, {
							get data() {
								return data_1();
							},
							get form() {
								return $$props.form;
							},
							get params() {
								return $$props.page.params;
							}
						}), ($$value) => components()[1] = $$value, () => components()?.[1]);
					});
					append($$anchor, fragment_2);
				},
				$$slots: { default: true }
			}), ($$value) => components()[0] = $$value, () => components()?.[0]);
		});
		append($$anchor, fragment_1);
	};
	var alternate = ($$anchor) => {
		const Pyramid_0 = user_derived(() => $$props.constructors[0]);
		var fragment_3 = comment();
		component(first_child(fragment_3), () => get(Pyramid_0), ($$anchor, Pyramid_0_2) => {
			bind_this(Pyramid_0_2($$anchor, {
				get data() {
					return data_0();
				},
				get form() {
					return $$props.form;
				},
				get params() {
					return $$props.page.params;
				}
			}), ($$value) => components()[0] = $$value, () => components()?.[0]);
		});
		append($$anchor, fragment_3);
	};
	if_block(node, ($$render) => {
		if ($$props.constructors[1]) $$render(consequent);
		else $$render(alternate, false);
	});
	var node_4 = sibling(node, 2);
	var consequent_2 = ($$anchor) => {
		var div = root_4();
		var node_5 = child(div);
		var consequent_1 = ($$anchor) => {
			var text$1 = text();
			template_effect(() => set_text(text$1, get(title)));
			append($$anchor, text$1);
		};
		if_block(node_5, ($$render) => {
			if (get(navigated)) $$render(consequent_1);
		});
		reset(div);
		append($$anchor, div);
	};
	if_block(node_4, ($$render) => {
		if (get(mounted)) $$render(consequent_2);
	});
	append($$anchor, fragment);
	pop();
}
var root_default = asClassComponent(Root);
const nodes = [
	() => __vitePreload(() => import("../nodes/0.Iy9pUNgx.js"), __vite__mapDeps([0,1,2,3]), import.meta.url),
	() => __vitePreload(() => import("../nodes/1.DqYHtEL4.js"), __vite__mapDeps([4,1,2,3]), import.meta.url),
	() => __vitePreload(() => import("../nodes/2.7sBeelal.js"), __vite__mapDeps([5,1,2,3]), import.meta.url)
];
const server_loads = [];
const dictionary = { "/": [2] };
const hooks = {
	handleError: (({ error }) => {
		console.error(error);
	}),
	reroute: (() => {}),
	transport: {}
};
const decoders = Object.fromEntries(Object.entries(hooks.transport).map(([k, v]) => [k, v.decode]));
const encoders = Object.fromEntries(Object.entries(hooks.transport).map(([k, v]) => [k, v.encode]));
const hash = false;
const decode = (type, value) => decoders[type](value);
var _layout_exports = /* @__PURE__ */ __exportAll({
	csr: () => true,
	load: () => load,
	prerender: () => false,
	ssr: () => false,
	trailingSlash: () => trailingSlash
}, 1);
const trailingSlash = "ignore";
var localHosts = [
	"localhost",
	"127.0.0.1",
	"sveltekit-prerender"
];
async function load({ url }) {
	globalThis._isLocal = localHosts.some((x) => url.host.includes(x));
	console.log("Env is local? = ", globalThis._isLocal, "|", url.host);
	console.log("obteniendo productos 1...");
	const productos = await getProductos(void 0);
	console.log("productos obtenidos:", productos.productos?.length);
	return { productos };
}
var blurhash_default = "(() => {\n  function thumbHashToRGBA(hash) {\n    let { PI, min, max, cos, round } = Math;\n\n    // Read the constants\n    let header24 = hash[0] | (hash[1] << 8) | (hash[2] << 16);\n    let header16 = hash[3] | (hash[4] << 8);\n    let l_dc = (header24 & 63) / 63;\n    let p_dc = ((header24 >> 6) & 63) / 31.5 - 1;\n    let q_dc = ((header24 >> 12) & 63) / 31.5 - 1;\n    let l_scale = ((header24 >> 18) & 31) / 31;\n    let hasAlpha = header24 >> 23;\n    let p_scale = ((header16 >> 3) & 63) / 63;\n    let q_scale = ((header16 >> 9) & 63) / 63;\n    let isLandscape = header16 >> 15;\n    let lx = max(3, isLandscape ? (hasAlpha ? 5 : 7) : header16 & 7);\n    let ly = max(3, isLandscape ? header16 & 7 : hasAlpha ? 5 : 7);\n    let a_dc = hasAlpha ? (hash[5] & 15) / 15 : 1;\n    let a_scale = (hash[5] >> 4) / 15;\n\n    // Read the varying factors (boost saturation by 1.25x to compensate for quantization)\n    let ac_start = hasAlpha ? 6 : 5;\n    let ac_index = 0;\n    let decodeChannel = (nx, ny, scale) => {\n      let ac = [];\n      for (let cy = 0; cy < ny; cy++)\n        for (let cx = cy ? 0 : 1; cx * ny < nx * (ny - cy); cx++)\n          ac.push(\n            (((hash[ac_start + (ac_index >> 1)] >> ((ac_index++ & 1) << 2)) &\n              15) /\n              7.5 -\n              1) *\n              scale,\n          );\n      return ac;\n    };\n    let l_ac = decodeChannel(lx, ly, l_scale);\n    let p_ac = decodeChannel(3, 3, p_scale * 1.25);\n    let q_ac = decodeChannel(3, 3, q_scale * 1.25);\n    let a_ac = hasAlpha && decodeChannel(5, 5, a_scale);\n\n    // Decode using the DCT into RGB\n    let ratio = thumbHashToApproximateAspectRatio(hash);\n    let w = round(ratio > 1 ? 32 : 32 * ratio);\n    let h = round(ratio > 1 ? 32 / ratio : 32);\n    let rgba = new Uint8Array(w * h * 4),\n      fx = [],\n      fy = [];\n    for (let y = 0, i = 0; y < h; y++) {\n      for (let x = 0; x < w; x++, i += 4) {\n        let l = l_dc,\n          p = p_dc,\n          q = q_dc,\n          a = a_dc;\n\n        // Precompute the coefficients\n        for (let cx = 0, n = max(lx, hasAlpha ? 5 : 3); cx < n; cx++)\n          fx[cx] = cos((PI / w) * (x + 0.5) * cx);\n        for (let cy = 0, n = max(ly, hasAlpha ? 5 : 3); cy < n; cy++)\n          fy[cy] = cos((PI / h) * (y + 0.5) * cy);\n\n        // Decode L\n        for (let cy = 0, j = 0; cy < ly; cy++)\n          for (\n            let cx = cy ? 0 : 1, fy2 = fy[cy] * 2;\n            cx * ly < lx * (ly - cy);\n            cx++, j++\n          )\n            l += l_ac[j] * fx[cx] * fy2;\n\n        // Decode P and Q\n        for (let cy = 0, j = 0; cy < 3; cy++) {\n          for (let cx = cy ? 0 : 1, fy2 = fy[cy] * 2; cx < 3 - cy; cx++, j++) {\n            let f = fx[cx] * fy2;\n            p += p_ac[j] * f;\n            q += q_ac[j] * f;\n          }\n        }\n\n        // Decode A\n        if (hasAlpha)\n          for (let cy = 0, j = 0; cy < 5; cy++)\n            for (let cx = cy ? 0 : 1, fy2 = fy[cy] * 2; cx < 5 - cy; cx++, j++)\n              a += a_ac[j] * fx[cx] * fy2;\n\n        // Convert to RGB\n        let b = l - (2 / 3) * p;\n        let r = (3 * l - b + q) / 2;\n        let g = r - q;\n        rgba[i] = max(0, 255 * min(1, r));\n        rgba[i + 1] = max(0, 255 * min(1, g));\n        rgba[i + 2] = max(0, 255 * min(1, b));\n        rgba[i + 3] = max(0, 255 * min(1, a));\n      }\n    }\n    return { w, h, rgba };\n  }\n\n  /**\n   * Extracts the average color from a ThumbHash. RGB is not be premultiplied by A.\n   *\n   * @param hash The bytes of the ThumbHash.\n   * @returns The RGBA values for the average color. Each value ranges from 0 to 1.\n   */\n  function thumbHashToAverageRGBA(hash) {\n    let { min, max } = Math;\n    let header = hash[0] | (hash[1] << 8) | (hash[2] << 16);\n    let l = (header & 63) / 63;\n    let p = ((header >> 6) & 63) / 31.5 - 1;\n    let q = ((header >> 12) & 63) / 31.5 - 1;\n    let hasAlpha = header >> 23;\n    let a = hasAlpha ? (hash[5] & 15) / 15 : 1;\n    let b = l - (2 / 3) * p;\n    let r = (3 * l - b + q) / 2;\n    let g = r - q;\n    return {\n      r: max(0, min(1, r)),\n      g: max(0, min(1, g)),\n      b: max(0, min(1, b)),\n      a,\n    };\n  }\n\n  /**\n   * Extracts the approximate aspect ratio of the original image.\n   *\n   * @param hash The bytes of the ThumbHash.\n   * @returns The approximate aspect ratio (i.e. width / height).\n   */\n  function thumbHashToApproximateAspectRatio(hash) {\n    let header = hash[3];\n    let hasAlpha = hash[2] & 0x80;\n    let isLandscape = hash[4] & 0x80;\n    let lx = isLandscape ? (hasAlpha ? 5 : 7) : header & 7;\n    let ly = isLandscape ? header & 7 : hasAlpha ? 5 : 7;\n    return lx / ly;\n  }\n\n  /**\n   * Encodes an RGBA image to a PNG data URL. RGB should not be premultiplied by\n   * A. This is optimized for speed and simplicity and does not optimize for size\n   * at all. This doesn't do any compression (all values are stored uncompressed).\n   *\n   * @param w The width of the input image. Must be ≤100px.\n   * @param h The height of the input image. Must be ≤100px.\n   * @param rgba The pixels in the input image, row-by-row. Must have w*h*4 elements.\n   * @returns A data URL containing a PNG for the input image.\n   */\n  function rgbaToDataURL(w, h, rgba) {\n    let row = w * 4 + 1;\n    let idat = 6 + h * (5 + row);\n    let bytes = [\n      137,\n      80,\n      78,\n      71,\n      13,\n      10,\n      26,\n      10,\n      0,\n      0,\n      0,\n      13,\n      73,\n      72,\n      68,\n      82,\n      0,\n      0,\n      w >> 8,\n      w & 255,\n      0,\n      0,\n      h >> 8,\n      h & 255,\n      8,\n      6,\n      0,\n      0,\n      0,\n      0,\n      0,\n      0,\n      0,\n      idat >>> 24,\n      (idat >> 16) & 255,\n      (idat >> 8) & 255,\n      idat & 255,\n      73,\n      68,\n      65,\n      84,\n      120,\n      1,\n    ];\n    let table = [\n      0, 498536548, 997073096, 651767980, 1994146192, 1802195444, 1303535960,\n      1342533948, -306674912, -267414716, -690576408, -882789492, -1687895376,\n      -2032938284, -1609899400, -1111625188,\n    ];\n    let a = 1,\n      b = 0;\n    for (let y = 0, i = 0, end = row - 1; y < h; y++, end += row - 1) {\n      bytes.push(\n        y + 1 < h ? 0 : 1,\n        row & 255,\n        row >> 8,\n        ~row & 255,\n        (row >> 8) ^ 255,\n        0,\n      );\n      for (b = (b + a) % 65521; i < end; i++) {\n        let u = rgba[i] & 255;\n        bytes.push(u);\n        a = (a + u) % 65521;\n        b = (b + a) % 65521;\n      }\n    }\n    bytes.push(\n      b >> 8,\n      b & 255,\n      a >> 8,\n      a & 255,\n      0,\n      0,\n      0,\n      0,\n      0,\n      0,\n      0,\n      0,\n      73,\n      69,\n      78,\n      68,\n      174,\n      66,\n      96,\n      130,\n    );\n    for (let [start, end] of [\n      [12, 29],\n      [37, 41 + idat],\n    ]) {\n      let c = ~0;\n      for (let i = start; i < end; i++) {\n        c ^= bytes[i];\n        c = (c >>> 4) ^ table[c & 15];\n        c = (c >>> 4) ^ table[c & 15];\n      }\n      c = ~c;\n      bytes[end++] = c >>> 24;\n      bytes[end++] = (c >> 16) & 255;\n      bytes[end++] = (c >> 8) & 255;\n      bytes[end++] = c & 255;\n    }\n    return \"data:image/png;base64,\" + btoa(String.fromCharCode(...bytes));\n  }\n\n  /**\n   * Decodes a ThumbHash to a PNG data URL. This is a convenience function that\n   * just calls \"thumbHashToRGBA\" followed by \"rgbaToDataURL\".\n   *\n   * @param hash The bytes of the ThumbHash.\n   * @returns A data URL containing a PNG for the rendered ThumbHash.\n   */\n\n  function base64ToUint8Array(base64) {\n    const binaryString = atob(base64);\n    return Uint8Array.from(binaryString, (char) => char.charCodeAt(0));\n  }\n\n  function thumbHashToDataURL(base64Hash) {\n    const hashBuffer = base64ToUint8Array(base64Hash);\n    let image = thumbHashToRGBA(hashBuffer);\n    return rgbaToDataURL(image.w, image.h, image.rgba);\n  }\n\n  const origins = {\n    \"1\": \"d16qwm950j0pjf.cloudfront.net/img-productos\"\n  }\n\n	const onload = () => {\n\n    for(const img of Array.from(document.getElementsByTagName(\"img\"))){\n      const role = img.getAttribute(\"role\")\n      if(role && role[1] === \"/\"){\n        const [origin, size, image] = role.split(\"/\")\n        if(origin === \"0\"){\n          img.src = thumbHashToDataURL(image)\n        } else if(origin === \"1\"){\n          const imageFull = image + (size ? `-x${size}.avif` : \"\")\n					img.src = \"https://\" + origins[origin] + \"/\" + imageFull\n					console.log(\"image source::\", img.src )\n        }\n      }\n    }\n\n    const subheader0 = document.getElementById(\"sh-0\")\n\n    const handleScroll = () => {\n      // console.log(\"comparison:\", window.scrollY, subheaderElement.offsetTop);\n\n      if (window.scrollY > (subheader0?.offsetTop || 0)) {\n        subheader0?.classList.add(\"s1\")\n      } else {\n        subheader0?.classList.remove(\"s1\")\n      }\n    };\n\n    window.addEventListener(\"scroll\", handleScroll);\n\n    return () => {\n      window.removeEventListener(\"scroll\", handleScroll);\n    };\n  }\n\n  if (typeof window != \"undefined\") {\n    document.addEventListener(\"DOMContentLoaded\", onload);\n  }\n\n  globalThis._makeThumbHashToDataURL = (hash) => thumbHashToDataURL(hash);\n})();\n";
var root_1$1 = from_html(`<!> <!>`, 1);
function _layout($$anchor, $$props) {
	push($$props, true);
	productosServiceState.categorias = $$props.data.productos.categorias;
	productosServiceState.productos = $$props.data.productos.productos;
	var fragment_1 = comment();
	head("q8odwi", ($$anchor) => {
		var fragment = root_1$1();
		var node = first_child(fragment);
		html(node, () => "<script>" + blurhash_default + "<\/script>");
		html(sibling(node, 2), () => `<link rel="stylesheet" href="libs/fontello-embedded.css">`);
		append($$anchor, fragment);
	});
	snippet(first_child(fragment_1), () => $$props.children);
	append($$anchor, fragment_1);
	pop();
}
var root$3 = from_html(`<div class="grid grid-cols-1 gap-8 p-2 md:grid-cols-2"><div class="mt-4"></div> <!> <!></div>`);
function LoginForm($$anchor, $$props) {
	let form = {
		usuario: "",
		password: ""
	};
	var div = root$3();
	var node = sibling(child(div), 2);
	Input(node, {
		get saveOn() {
			return form;
		},
		css: "mb-2",
		save: "usuario",
		label: "Usuario"
	});
	Input(sibling(node, 2), {
		get saveOn() {
			return form;
		},
		css: "mb-2",
		type: "password",
		save: "password",
		label: "Password"
	});
	reset(div);
	append($$anchor, div);
}
var root$2 = from_html(`<div id="mob-layer"><div class="flex items-center justify-between absolute w-full top-9 pl-20 pr-4"><div class="fs18 text-[white]"> </div> <button class="_4 fs20 border-none outline-none text-[white] bg-transparent"><i class="icon-cancel"></i></button></div> <div class="_3 w-full absolute bottom-0 afc"><!></div></div>`);
function MobileLayerVertical($$anchor, $$props) {
	push($$props, true);
	const id = prop($$props, "id", 3, false), title = prop($$props, "title", 3, "");
	let divContainer;
	const show = user_derived(() => Core.openLayers.includes(id()));
	user_effect(() => {
		if (get(show)) suscribeUrlFlag("mob-layer", () => {
			Core.openLayers = Core.openLayers.filter((x) => x !== id());
		});
	});
	let showInner = state(proxy(get(show)));
	user_effect(() => {
		if (typeof get(show) !== "undefined" && typeof document !== "undefined") {
			divContainer.style.setProperty("view-transition-name", "mobile-layer-vertical");
			setTimeout(() => {
				divContainer.style.setProperty("view-transition-name", "");
			}, 400);
			if (document.startViewTransition) document.startViewTransition(() => {
				set(showInner, get(show), true);
			});
			else set(showInner, get(show), true);
		}
	});
	let css = user_derived(() => {
		let value = "_1";
		if (get(showInner)) value += " _2";
		return value;
	});
	var div = root$2();
	var div_1 = child(div);
	var div_2 = child(div_1);
	var text = child(div_2, true);
	reset(div_2);
	var button = sibling(div_2, 2);
	button.__click = (ev) => {
		ev.stopPropagation();
		Core.openLayers = Core.openLayers.filter((x) => x !== id());
	};
	reset(div_1);
	var div_3 = sibling(div_1, 2);
	slot(child(div_3), $$props, "default", {}, null);
	reset(div_3);
	reset(div);
	bind_this(div, ($$value) => divContainer = $$value, () => divContainer);
	template_effect(() => {
		set_class(div, 1, get(css) + " top-10 left-0 fixed w-[100vw]", "afc");
		set_attribute(div, "aria-hidden", !get(showInner));
		set_text(text, title());
	});
	append($$anchor, div);
	pop();
}
delegate(["click"]);
var root_1 = from_html(`<div class="_7 aeU"><div class="_8 aeU"></div> <div class="h-24 fs20 mt-[-4px]"><i></i></div> <div class="flex items-center text-center grow-1"> </div></div>`);
var root$1 = from_html(`<div></div> <div id="mob-menu"><button class="_3 absolute top-4 right-4 w-40 h-40 aeU"><i class="icon-cancel"></i></button> <div class="grid gap-8 grid-cols-2 p-8 mt-54"></div></div> <!>`, 1);
function MobileMenu($$anchor, $$props) {
	push($$props, true);
	let divContainer;
	const closeMenu = (event) => {
		event?.stopPropagation?.();
		divContainer.style.setProperty("view-transition-name", "mobile-side-menu");
		setTimeout(() => {
			divContainer.style.setProperty("view-transition-name", "");
		}, 400);
		if (document.startViewTransition) document.startViewTransition(() => {
			Core.mobileMenuOpen = 0;
		});
		else Core.mobileMenuOpen = 0;
	};
	user_effect(() => {
		if (Core.mobileMenuOpen) suscribeUrlFlag("mob-menu", () => {
			closeMenu();
		});
	});
	let css = user_derived(() => {
		let css = "_1 w-[74vw] h-[100vh] fixed top-0 left-0";
		if (Core.mobileMenuOpen === 1) {
			divContainer.style.setProperty("view-transition-name", "mobile-side-menu");
			css += " _2";
			setTimeout(() => {
				divContainer.style.setProperty("view-transition-name", "");
			}, 400);
		}
		return css;
	});
	let overlayCss = user_derived(() => {
		let css = "_4 top-0 left-0 fixed w-[100vw] h-[100vh]";
		if (Core.mobileMenuOpen) css += " _5";
		return css;
	});
	var fragment = root$1();
	var div = first_child(fragment);
	div.__click = closeMenu;
	var div_1 = sibling(div, 2);
	var button = child(div_1);
	button.__click = closeMenu;
	var div_2 = sibling(button, 2);
	each(div_2, 21, () => mainMenuOptions, index, ($$anchor, opt) => {
		var div_3 = root_1();
		div_3.__click = (ev) => {
			ev.stopPropagation();
			if (get(opt).onClick) get(opt).onClick();
		};
		var div_4 = sibling(child(div_3), 2);
		var i = child(div_4);
		reset(div_4);
		var div_5 = sibling(div_4, 2);
		var text = child(div_5, true);
		reset(div_5);
		reset(div_3);
		template_effect(() => {
			set_class(i, 1, clsx(get(opt).icon), "aeU");
			set_text(text, get(opt).name);
		});
		append($$anchor, div_3);
	});
	reset(div_2);
	reset(div_1);
	bind_this(div_1, ($$value) => divContainer = $$value, () => divContainer);
	MobileLayerVertical(sibling(div_1, 2), {
		title: "Iniciar Sesión",
		id: 1,
		children: ($$anchor, $$slotProps) => {
			LoginForm($$anchor, {});
		},
		$$slots: { default: true }
	});
	template_effect(() => {
		set_class(div, 1, clsx(get(overlayCss)), "aeU");
		set_class(div_1, 1, clsx(get(css)), "aeU");
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click"]);
var root = from_html(`<!> <!> <!> <div class="h-800 _1 aeT"><div>hola</div> <!></div> <div class="h-800 _1 flex aeT"></div>`, 1);
function _page($$anchor) {
	let categorias = [
		{
			Name: "Perfumes",
			Image: "images/categoria_11.webp"
		},
		{
			Name: "Casacas",
			Image: "images/uwgGHwD2aYl3.webp"
		},
		{
			Name: "Zapatos",
			Image: "images/MqcNHwj3iKeK.webp"
		},
		{
			Name: "Relojes",
			Image: "images/7ogGDobEysil.webp"
		},
		{
			Name: "Decoración",
			Image: "images/lHSFC4IPj2mN.webp"
		}
	];
	var fragment = root();
	var node = first_child(fragment);
	Header(node, {});
	var node_1 = sibling(node, 2);
	MobileMenu(node_1, {});
	var node_2 = sibling(node_1, 2);
	MainCarrusel(node_2, { get categorias() {
		return categorias;
	} });
	var div = sibling(node_2, 2);
	ProductCards(sibling(child(div), 2), {});
	reset(div);
	next(2);
	append($$anchor, fragment);
}
export { decoders as a, hash as c, server_loads as d, root_default as f, decode as i, hooks as l, _layout as n, dictionary as o, matchers as p, _layout_exports as r, encoders as s, _page as t, nodes as u };
