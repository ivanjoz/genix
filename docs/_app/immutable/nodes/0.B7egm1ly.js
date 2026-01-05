import { $ as proxy, A as if_block, B as tick, C as clsx, E as snippet, G as user_effect, I as head, J as $document, L as delegate, M as append, N as comment, O as each, P as from_html, Q as sibling, R as event, S as set_class, St as __exportAll, Tt as __toESM, V as untrack, W as template_effect, X as child, Z as first_child, _t as reset, a as prop, at as state, bt as false_default, ct as getContext, f as bind_this, j as set_text, k as index, lt as pop, o as setup_stores, r as onMount, rt as set, s as store_get, st as user_derived, ut as push, v as set_attribute, vt as noop, x as set_style, z as get } from "../chunks/CTB4HzdN.js";
import { n as true_default } from "../chunks/D9PGjZ6Y.js";
import { r as stores, t as goto } from "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { t as browser } from "../chunks/BemFyt0E.js";
import { t as page$1 } from "../chunks/Dm_jeqTV.js";
import { _ as doInitServiceWorker, a as fetchOnCourse, d as checkIsLogin, o as getDeviceType, t as Core, u as accessHelper, y as Env } from "../chunks/BwrZ3UQO.js";
import { i as parseSVG, o as require_notiflix_aio_3_2_8_min } from "../chunks/DwKmqXYU.js";
import { l as throttle, o as highlString, s as include } from "../chunks/DOFkf9MZ.js";
import { t as angle_default } from "../chunks/B0DNpKVT.js";
import { t as Input } from "../chunks/N4ddDIuf.js";
import { t as OptionsStrip } from "../chunks/CGFlWCTf.js";
import { i as postUsuarioPropio } from "../chunks/BKsRj9OC.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as modules_default } from "../chunks/CoH7Ebne.js";
var _layout_exports = /* @__PURE__ */ __exportAll({ ssr: () => false }, 1);
var root_2$3 = from_html(`<span> </span>`);
var root_1$5 = from_html(`<div class="_7 aMu"><div class="w-full"></div></div>`);
var root$5 = from_html(`<div><div class="flex items-center p-6"><i class="icon-search _3 aMu"></i> <textarea class="h-38 _4 aMu">
    </textarea> <button class="ml-auto h-40 w-40 shrink-0 _5 aMu" aria-label="Buscar"><i class="icon-cancel h1"></i></button></div> <div class="_6 aMu"></div></div>`);
function TopLayerSelector($$anchor, $$props) {
	push($$props, true);
	const isOpen = user_derived(() => !!Core.showMobileSearchLayer);
	let avoidClose = false;
	let htmlTextarea;
	let searchText = state("");
	const keyName = user_derived(() => Core.showMobileSearchLayer?.keyName);
	user_derived(() => Core.showMobileSearchLayer?.keyID);
	const searchWords = user_derived(() => get(searchText).toLowerCase().split(" ").filter((x) => x.length > 1));
	const optionsFiltered = user_derived(() => {
		if ((get(searchText) || "").trim() === "") {
			console.log("Enviando options::", Core.showMobileSearchLayer?.options);
			return Core.showMobileSearchLayer?.options || [];
		} else {
			const filtered = [];
			for (const opt of Core.showMobileSearchLayer?.options || []) {
				const name = opt[get(keyName)];
				if (typeof name === "string") {
					if (include(name.toLowerCase(), get(searchWords))) filtered.push(opt);
				}
			}
			console.log("Enviando filtered::", get(searchText), filtered);
			return filtered;
		}
	});
	user_effect(() => {
		if (get(isOpen)) untrack(() => {
			if (htmlTextarea) {
				htmlTextarea.focus();
				htmlTextarea.value = "";
			}
			set(searchText, "");
		});
		else if (htmlTextarea) {
			htmlTextarea.blur();
			htmlTextarea.value = "";
		}
	});
	var div = root$5();
	let classes;
	div.__mousedown = (ev) => {
		if (ev.target.tagName === "textarea") return;
		avoidClose = true;
	};
	var div_1 = child(div);
	var textarea = sibling(child(div_1), 2);
	set_attribute(textarea, "rows", 1);
	textarea.__keyup = (ev) => {
		throttle(() => {
			set(searchText, (ev.target.value || "").toLowerCase(), true);
		}, 150);
	};
	bind_this(textarea, ($$value) => htmlTextarea = $$value, () => htmlTextarea);
	var button = sibling(textarea, 2);
	button.__click = (ev) => {
		Core.showMobileSearchLayer = null;
	};
	reset(div_1);
	var div_2 = sibling(div_1, 2);
	each(div_2, 21, () => get(optionsFiltered), index, ($$anchor$1, opt) => {
		var div_3 = root_1$5();
		div_3.__click = () => {
			Core.showMobileSearchLayer?.onSelect(get(opt));
			Core.showMobileSearchLayer = null;
		};
		var div_4 = child(div_3);
		each(div_4, 21, () => highlString(get(opt)[get(keyName)], get(searchWords)), index, ($$anchor$2, w) => {
			var span = root_2$3();
			var text = child(span, true);
			reset(span);
			template_effect(() => {
				set_class(span, 1, clsx(get(w).highl ? "_8" : ""), "aMu");
				set_text(text, get(w).text);
			});
			append($$anchor$2, span);
		});
		reset(div_4);
		reset(div_3);
		append($$anchor$1, div_3);
	});
	reset(div_2);
	reset(div);
	template_effect(() => classes = set_class(div, 1, "_1 aMu", null, classes, { _2: get(isOpen) }));
	event("blur", textarea, () => {
		if (avoidClose) {
			avoidClose = false;
			htmlTextarea?.focus();
		} else Core.showMobileSearchLayer = null;
	});
	append($$anchor, div);
	pop();
}
delegate([
	"mousedown",
	"keyup",
	"click"
]);
var favicon_default = "data:image/svg+xml,%3csvg%20xmlns='http://www.w3.org/2000/svg'%20width='107'%20height='128'%20viewBox='0%200%20107%20128'%3e%3ctitle%3esvelte-logo%3c/title%3e%3cpath%20d='M94.157%2022.819c-10.4-14.885-30.94-19.297-45.792-9.835L22.282%2029.608A29.92%2029.92%200%200%200%208.764%2049.65a31.5%2031.5%200%200%200%203.108%2020.231%2030%2030%200%200%200-4.477%2011.183%2031.9%2031.9%200%200%200%205.448%2024.116c10.402%2014.887%2030.942%2019.297%2045.791%209.835l26.083-16.624A29.92%2029.92%200%200%200%2098.235%2078.35a31.53%2031.53%200%200%200-3.105-20.232%2030%2030%200%200%200%204.474-11.182%2031.88%2031.88%200%200%200-5.447-24.116'%20style='fill:%23ff3e00'/%3e%3cpath%20d='M45.817%20106.582a20.72%2020.72%200%200%201-22.237-8.243%2019.17%2019.17%200%200%201-3.277-14.503%2018%2018%200%200%201%20.624-2.435l.49-1.498%201.337.981a33.6%2033.6%200%200%200%2010.203%205.098l.97.294-.09.968a5.85%205.85%200%200%200%201.052%203.878%206.24%206.24%200%200%200%206.695%202.485%205.8%205.8%200%200%200%201.603-.704L69.27%2076.28a5.43%205.43%200%200%200%202.45-3.631%205.8%205.8%200%200%200-.987-4.371%206.24%206.24%200%200%200-6.698-2.487%205.7%205.7%200%200%200-1.6.704l-9.953%206.345a19%2019%200%200%201-5.296%202.326%2020.72%2020.72%200%200%201-22.237-8.243%2019.17%2019.17%200%200%201-3.277-14.502%2017.99%2017.99%200%200%201%208.13-12.052l26.081-16.623a19%2019%200%200%201%205.3-2.329%2020.72%2020.72%200%200%201%2022.237%208.243%2019.17%2019.17%200%200%201%203.277%2014.503%2018%2018%200%200%201-.624%202.435l-.49%201.498-1.337-.98a33.6%2033.6%200%200%200-10.203-5.1l-.97-.294.09-.968a5.86%205.86%200%200%200-1.052-3.878%206.24%206.24%200%200%200-6.696-2.485%205.8%205.8%200%200%200-1.602.704L37.73%2051.72a5.42%205.42%200%200%200-2.449%203.63%205.79%205.79%200%200%200%20.986%204.372%206.24%206.24%200%200%200%206.698%202.486%205.8%205.8%200%200%200%201.602-.704l9.952-6.342a19%2019%200%200%201%205.295-2.328%2020.72%2020.72%200%200%201%2022.237%208.242%2019.17%2019.17%200%200%201%203.277%2014.503%2018%2018%200%200%201-8.13%2012.053l-26.081%2016.622a19%2019%200%200%201-5.3%202.328'%20style='fill:%23fff'/%3e%3c/svg%3e";
var root_1$4 = from_html(`<button type="button"><!></button>`);
var root_2$2 = from_html(`<button type="button"> </button>`);
var root_3$2 = from_html(`<div><div class="aKY aN6"><img class="aKZ aN6" alt=""/></div> <div><!></div></div>`);
var root$4 = from_html(`<div class="aKU aN6"><!> <!></div>`);
function ButtonLayer($$anchor, $$props) {
	push($$props, true);
	let buttonText = prop($$props, "buttonText", 3, "Open"), buttonClass = prop($$props, "buttonClass", 3, ""), layerClass = prop($$props, "layerClass", 3, ""), defaultOpen = prop($$props, "defaultOpen", 3, false), contentCss = prop($$props, "contentCss", 3, "");
	let isOpen = state(proxy(defaultOpen()));
	let buttonElement = state(null);
	let layerElement = state(null);
	let position = state(proxy({
		top: 0,
		left: 0
	}));
	let angleLeft = state(20);
	function toggleLayer() {
		set(isOpen, !get(isOpen));
		if (get(isOpen)) $$props.onOpen?.();
		else $$props.onClose?.();
	}
	function closeLayer() {
		if (get(isOpen)) {
			set(isOpen, false);
			$$props.onClose?.();
		}
	}
	async function updatePosition() {
		if (!get(buttonElement) || !get(layerElement)) return;
		await tick();
		const buttonRect = get(buttonElement).getBoundingClientRect();
		const layerRect = get(layerElement).getBoundingClientRect();
		const isMobile = window.innerWidth <= 748;
		const offset = 8;
		let top = buttonRect.bottom + offset;
		let left = isMobile ? 6 : buttonRect.left;
		if (!isMobile) {
			if (left + layerRect.width > window.innerWidth) left = window.innerWidth - layerRect.width - 10;
			if (left < 10) left = 10;
		}
		if (top + layerRect.height > window.innerHeight) top = buttonRect.top - layerRect.height - offset;
		set(position, {
			top,
			left
		}, true);
		const buttonCenter = buttonRect.left + buttonRect.width / 2;
		if (isMobile) {
			set(angleLeft, buttonCenter - left - 12);
			const layerWidth = window.innerWidth - 12;
			const minAngleLeft = 10;
			const maxAngleLeft = layerWidth - 24;
			if (get(angleLeft) < minAngleLeft) set(angleLeft, minAngleLeft);
			if (get(angleLeft) > maxAngleLeft) set(angleLeft, maxAngleLeft);
		} else {
			set(angleLeft, buttonCenter - left - 12);
			const minAngleLeft = 10;
			const maxAngleLeft = layerRect.width - 24;
			if (get(angleLeft) < minAngleLeft) set(angleLeft, minAngleLeft);
			if (get(angleLeft) > maxAngleLeft) set(angleLeft, maxAngleLeft);
		}
	}
	user_effect(() => {
		if (get(isOpen) && get(buttonElement) && get(layerElement)) updatePosition();
	});
	user_effect(() => {
		if (!get(isOpen)) return;
		function handleClickOutside(event$1) {
			const target = event$1.target;
			if (get(buttonElement) && !get(buttonElement).contains(target) && get(layerElement) && !get(layerElement).contains(target)) closeLayer();
		}
		document.body.addEventListener("mousedown", handleClickOutside);
		return () => {
			document.body.removeEventListener("mousedown", handleClickOutside);
		};
	});
	user_effect(() => {
		if (!get(isOpen)) return;
		const handleUpdate = () => {
			if (get(isOpen) && get(buttonElement) && get(layerElement)) updatePosition();
		};
		window.addEventListener("scroll", handleUpdate, true);
		window.addEventListener("resize", handleUpdate);
		return () => {
			window.removeEventListener("scroll", handleUpdate, true);
			window.removeEventListener("resize", handleUpdate);
		};
	});
	var div = root$4();
	var node = child(div);
	var consequent = ($$anchor$1) => {
		var button_1 = root_1$4();
		button_1.__click = toggleLayer;
		snippet(child(button_1), () => $$props.button);
		reset(button_1);
		bind_this(button_1, ($$value) => set(buttonElement, $$value), () => get(buttonElement));
		template_effect(() => set_class(button_1, 1, buttonClass(), "aN6"));
		append($$anchor$1, button_1);
	};
	var alternate = ($$anchor$1) => {
		var button_2 = root_2$2();
		button_2.__click = toggleLayer;
		var text = child(button_2, true);
		reset(button_2);
		bind_this(button_2, ($$value) => set(buttonElement, $$value), () => get(buttonElement));
		template_effect(() => {
			set_class(button_2, 1, `aKV ${buttonClass() ?? ""}`, "aN6");
			set_text(text, buttonText());
		});
		append($$anchor$1, button_2);
	};
	if_block(node, ($$render) => {
		if ($$props.button) $$render(consequent);
		else $$render(alternate, false);
	});
	var node_2 = sibling(node, 2);
	var consequent_1 = ($$anchor$1) => {
		var div_1 = root_3$2();
		var div_2 = child(div_1);
		var img = child(div_2);
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		snippet(child(div_3), () => $$props.children ?? noop);
		reset(div_3);
		reset(div_1);
		bind_this(div_1, ($$value) => set(layerElement, $$value), () => get(layerElement));
		template_effect(($0) => {
			set_class(div_1, 1, `aKW min-w-200 w-[calc(100vw-12px)] ${layerClass() ?? ""}`, "aN6");
			set_style(div_1, `top: ${get(position).top ?? ""}px; left: ${get(position).left ?? ""}px;`);
			set_style(div_2, `left: ${get(angleLeft) ?? ""}px;`);
			set_attribute(img, "src", $0);
			set_class(div_3, 1, `aKX ${contentCss() ?? ""}`, "aN6");
		}, [() => parseSVG(angle_default)]);
		append($$anchor$1, div_1);
	};
	if_block(node_2, ($$render) => {
		if (get(isOpen)) $$render(consequent_1);
	});
	reset(div);
	append($$anchor, div);
	pop();
}
delegate(["click"]);
var import_notiflix_aio_3_2_8_min = /* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min());
var root_1$3 = from_html(`<div class="w-full flex mb-12 mt-[-2px]"><div class="mr-auto"></div> <button class="bx-blue mr-12" aria-label="Guardar Usuario"><i class="icon-floppy"></i></button> <button class="bx-orange" aria-label="Salir"><i class="icon-logout-1"></i> <span>Salir</span></button></div> <div class="grid grid-cols-24 w-full gap-10"><!> <!> <!> <!> <!> <div class="col-span-24"><div class="ff-bold mb-[-4px] mt-2">Cambiar Password</div></div> <!> <!></div>`, 1);
var root$3 = from_html(`<div class="flex items-center"><!></div> <!>`, 1);
function HeaderConfig($$anchor, $$props) {
	push($$props, true);
	const options = [{
		id: 1,
		name: "Usuario"
	}, {
		id: 2,
		name: "Config."
	}];
	let selected = state(1);
	function handleLogout() {
		localStorage.clear();
		sessionStorage.clear();
		window.location.href = "/login";
	}
	let userInfo = state(proxy(accessHelper.getUserInfo()));
	user_effect(() => {
		if (get(selected) === 1) set(userInfo, set(userInfo, accessHelper.getUserInfo(), true), true);
	});
	const saveUsuario = async () => {
		if (get(userInfo).password1 && get(userInfo).password1 !== get(userInfo).password2) import_notiflix_aio_3_2_8_min.Notify.failure("Los password no coinciden.");
		import_notiflix_aio_3_2_8_min.Loading.standard("Creando/Actualizando Usuario...");
		try {
			var result = await postUsuarioPropio(get(userInfo));
		} catch (error) {
			import_notiflix_aio_3_2_8_min.Notify.failure(error);
			import_notiflix_aio_3_2_8_min.Loading.remove();
			return;
		}
		import_notiflix_aio_3_2_8_min.Loading.remove();
		accessHelper.setUserInfo(get(userInfo));
		console.log("usuario result::", result);
	};
	var fragment = root$3();
	var div = first_child(fragment);
	OptionsStrip(child(div), {
		get options() {
			return options;
		},
		keyId: "id",
		keyName: "name",
		get selected() {
			return get(selected);
		},
		onSelect: (e) => set(selected, e.id, true)
	});
	reset(div);
	var node_1 = sibling(div, 2);
	var consequent = ($$anchor$1) => {
		var fragment_1 = root_1$3();
		var div_1 = first_child(fragment_1);
		var button = sibling(child(div_1), 2);
		button.__click = () => {
			saveUsuario();
		};
		var button_1 = sibling(button, 2);
		button_1.__click = handleLogout;
		reset(div_1);
		var div_2 = sibling(div_1, 2);
		var node_2 = child(div_2);
		Input(node_2, {
			label: "Nombres",
			css: "col-span-12",
			get saveOn() {
				return get(userInfo);
			},
			save: "nombres"
		});
		var node_3 = sibling(node_2, 2);
		Input(node_3, {
			label: "Apellidos",
			css: "col-span-12",
			get saveOn() {
				return get(userInfo);
			},
			save: "apellidos"
		});
		var node_4 = sibling(node_3, 2);
		Input(node_4, {
			label: "Email",
			css: "col-span-12",
			get saveOn() {
				return get(userInfo);
			},
			save: "email"
		});
		var node_5 = sibling(node_4, 2);
		Input(node_5, {
			label: "Cargo",
			css: "col-span-12",
			get saveOn() {
				return get(userInfo);
			},
			save: "cargo"
		});
		var node_6 = sibling(node_5, 2);
		Input(node_6, {
			label: "Nº Documento",
			css: "col-span-12",
			get saveOn() {
				return get(userInfo);
			},
			save: "documentoNro"
		});
		var node_7 = sibling(node_6, 4);
		Input(node_7, {
			label: "Password",
			css: "col-span-12",
			get saveOn() {
				return get(userInfo);
			},
			save: "password1",
			type: "password"
		});
		Input(sibling(node_7, 2), {
			label: "Repetir Password",
			css: "col-span-12",
			get saveOn() {
				return get(userInfo);
			},
			save: "password2",
			type: "password"
		});
		reset(div_2);
		append($$anchor$1, fragment_1);
	};
	if_block(node_1, ($$render) => {
		if (get(selected) === 1) $$render(consequent);
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click"]);
var root_1$2 = from_html(`<button type="button" class="md:hidden p-8 hover:bg-white/10 rounded-lg transition-colors mr-12 cursor-pointer" aria-label="Toggle menu"><span class="text-white text-2xl">☰</span></button>`);
var root_3$1 = from_html(`<button> </button>`);
var root_4$2 = from_html(`<div class="h1 text-white text-lg font-semibold tracking-wide"> </div>`);
var root_5$1 = from_html(`<div class="aKr mr-06 aMx"><div class="bg aMx"></div> <span></span></div>`);
var root_6$1 = from_html(`<span class="text-white text-lg icon-cog"></span>`);
var root_8$1 = from_html(`<div class="fixed inset-0 z-30" role="button" tabindex="0" aria-label="Close settings"></div>`);
var root$2 = from_html(`<header class="_1 fixed top-0 left-0 right-0 bg-gradient-to-r from-indigo-600 to-indigo-700 
	shadow-md z-150 flex items-center px-6 md:px-16 aMx"><div class="hidden md:flex items-center justify-center h-full w-56 mr-12"><div class="h-40 w-40 bg-black/20 rounded-lg flex items-center justify-center"><img src="/images/genix_logo4.svg" alt="Genix Logo" class="w-full h-full p-1"/></div></div> <!> <div class="flex-1 flex items-center"><!></div> <div class="flex items-center gap-8 h-full relative"><!> <div class="relative"><!></div> <button aria-label="Reload"><span class="text-white text-lg icon-cw"></span></button></div></header> <!>`, 1);
function Header($$anchor, $$props) {
	push($$props, true);
	const showMenuButton = prop($$props, "showMenuButton", 3, false);
	let showSettings = state(false);
	let uiTheme = state("light");
	let isReloading = state(false);
	function handleReload() {
		set(isReloading, true);
		const now5secodsMore = Math.floor(Date.now() / 1e3) + 5;
		localStorage.setItem("force_sync_cache_until", String(now5secodsMore));
		window.location.reload();
	}
	user_effect(() => {
		const savedTheme = localStorage.getItem("ui-color") || "light";
		if (savedTheme === "dark" || savedTheme === "light") {
			set(uiTheme, savedTheme, true);
			document.body.classList.add(savedTheme);
		}
	});
	var fragment = root$2();
	var header = first_child(fragment);
	var node = sibling(child(header), 2);
	var consequent = ($$anchor$1) => {
		var button_1 = root_1$2();
		button_1.__click = () => Core.toggleMobileMenu();
		append($$anchor$1, button_1);
	};
	if_block(node, ($$render) => {
		if (showMenuButton()) $$render(consequent);
	});
	var div = sibling(node, 2);
	var node_1 = child(div);
	var consequent_1 = ($$anchor$1) => {
		var fragment_1 = comment();
		each(first_child(fragment_1), 17, () => Core.pageOptions, index, ($$anchor$2, opt) => {
			const selected = user_derived(() => Core.pageOptionSelected == get(opt).id);
			var button_2 = root_3$1();
			let classes;
			button_2.__click = () => {
				Core.pageOptionSelected = get(opt).id;
			};
			var text = child(button_2, true);
			reset(button_2);
			template_effect(() => {
				classes = set_class(button_2, 1, "_2 aMx", null, classes, { _3: get(selected) });
				set_attribute(button_2, "aria-label", get(opt).name);
				set_text(text, get(opt).name);
			});
			append($$anchor$2, button_2);
		});
		append($$anchor$1, fragment_1);
	};
	var alternate = ($$anchor$1) => {
		var div_1 = root_4$2();
		var text_1 = child(div_1, true);
		reset(div_1);
		template_effect(() => set_text(text_1, Core.pageTitle));
		append($$anchor$1, div_1);
	};
	if_block(node_1, ($$render) => {
		if (Core.pageOptions?.length > 0) $$render(consequent_1);
		else $$render(alternate, false);
	});
	reset(div);
	var div_2 = sibling(div, 2);
	var node_3 = child(div_2);
	var consequent_2 = ($$anchor$1) => {
		var div_3 = root_5$1();
		var span = sibling(child(div_3), 2);
		span.textContent = "Cargando...";
		reset(div_3);
		append($$anchor$1, div_3);
	};
	if_block(node_3, ($$render) => {
		if (fetchOnCourse.size > 0) $$render(consequent_2);
	});
	var div_4 = sibling(node_3, 2);
	var node_4 = child(div_4);
	{
		const button = ($$anchor$1) => {
			append($$anchor$1, root_6$1());
		};
		ButtonLayer(node_4, {
			layerClass: "md:w-640 md:h-460 px-8 py-6",
			buttonClass: "w-40 h-40 rounded-full bg-white/10 hover:bg-white/20 \n					flex items-center justify-center transition-colors shadow-sm",
			contentCss: "px-4 pb-8 md:px-8 md:py-8",
			button,
			children: ($$anchor$1, $$slotProps) => {
				HeaderConfig($$anchor$1, {});
			},
			$$slots: {
				button: true,
				default: true
			}
		});
	}
	reset(div_4);
	var button_3 = sibling(div_4, 2);
	button_3.__click = handleReload;
	reset(div_2);
	reset(header);
	var node_5 = sibling(header, 2);
	var consequent_3 = ($$anchor$1) => {
		var div_5 = root_8$1();
		div_5.__click = () => set(showSettings, false);
		div_5.__keydown = (e) => e.key === "Escape" && set(showSettings, false);
		append($$anchor$1, div_5);
	};
	if_block(node_5, ($$render) => {
		if (get(showSettings)) $$render(consequent_3);
	});
	template_effect(() => {
		set_class(button_3, 1, `hidden md:flex w-40 h-40 rounded-full bg-white/10 hover:bg-white/20 
				items-center justify-center transition-colors shadow-sm
				${get(isReloading) ? "aKq" : ""}`, "aMx");
		button_3.disabled = get(isReloading);
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click", "keydown"]);
const getStores = () => {
	const stores$1 = stores;
	return {
		page: { subscribe: stores$1.page.subscribe },
		navigating: { subscribe: stores$1.navigating.subscribe },
		updated: stores$1.updated
	};
};
const page = { subscribe(fn) {
	return getStores().page.subscribe(fn);
} };
var root_2$1 = from_html(`<span><i class="icon-down-open-1"></i></span>`);
var root_5 = from_html(`<span class="mr-4"> </span>`);
var root_6 = from_html(`<div class="absolute left-0 top-1/2 -translate-y-1/2 w-1 h-3/4 bg-indigo-400 rounded-r"></div>`);
var root_4$1 = from_html(`<button><div class="aKx flex w-full aMB"><i></i> <div class="font-mono aMB"> </div></div> <div class="aKy flex w-full aMB"><i></i> <div class="font-mono aMB"></div></div> <!></button>`);
var root_3 = from_html(`<div class="transition-all duration-200"></div>`);
var root_1$1 = from_html(`<div class="overflow-hidden transition-all duration-400 mb-1"><button><div class="flex items-center flex-1 min-w-0 whitespace-nowrap"><span class="aKu font-mono font-semibold ml-1 aMB"> </span> <span class="aKv font-mono font-semibold ml-1 tracking-wider aMB"> </span></div> <!></button> <!></div>`);
var root_8 = from_html(`<span><i class="icon-down-open-1"></i></span>`);
var root_11 = from_html(`<i></i>`);
var root_10 = from_html(`<button><!> <span class="aKT aMB"> </span></button>`);
var root_9 = from_html(`<div class="aKO aMB"></div>`);
var root_7 = from_html(`<div class="aKJ aMB"><button><span class="aKL aMB"> </span> <!></button> <!></div>`);
var root$1 = from_html(`<div class="aKt fixed left-0 top-0 h-screen bg-gradient-to-b from-gray-900 via-gray-900 to-gray-950 
		text-white shadow-xl transition-all duration-200 ease-in-out z-300 hidden md:block aMB" role="navigation" aria-label="Main navigation"><div class="flex items-center h-48 border-b border-gray-800/30 w-full"><div class="_1 flex items-center aMB"><img class="w-42 h-42" src="/images/genix_logo4.svg" alt=""/> <div class="_2 hidden white ff-bold h2 ml-[-3px] aMB">enix</div></div></div> <div class="flex-1 transition-all duration-300 w-full"></div></div> <div role="dialog" aria-modal="true"><button type="button" class="aKB aMB" aria-label="Close menu"></button> <aside class="aKD aMB"><div class="aKE h-48 flex items-center px-6 justify-between aMB"><div class="aKF aMB"><img src="/images/genix_logo3.svg" alt="Genix" class="size-36"/> <span class="aKG aMB">GENIX</span></div> <button class="aKH size-32 aMB" aria-label="Close menu"><i class="icon-cancel"></i></button></div> <div class="aKI aMB"></div></aside></div>`, 1);
function SideMenu($$anchor, $$props) {
	push($$props, true);
	const $page = () => store_get(page, "$page", $$stores);
	const [$$stores, $$cleanup] = setup_stores();
	const module = user_derived(() => Core.module);
	let menuOpen = state(proxy([0, ""]));
	let currentPathname = state("");
	let mobileMenuPanel;
	let mobileMenuBackdrop;
	function getMenuOpenFromRoute(mod, pathname) {
		for (let menu of mod.menus) for (let opt of menu.options || []) if (opt.route === pathname) return [menu.id || 0, opt.route];
		return [0, ""];
	}
	function toggleMenu(menuId) {
		if (get(menuOpen)[0] === menuId) set(menuOpen, [0, ""], true);
		else set(menuOpen, [menuId, get(menuOpen)[1]], true);
	}
	function applyActiveStylesInstant(button) {
		const activeButton = mobileMenuPanel?.querySelector(".mobile-menu-option.is-active");
		if (activeButton) activeButton.classList.remove("is-active");
		button.classList.add("is-active");
	}
	async function navigateTo(route, menuId, buttonElement) {
		if (Core.mobileMenuOpen && buttonElement) applyActiveStylesInstant(buttonElement);
		goto(route);
		if (Core.mobileMenuOpen) toggleMobileMenu(true);
		set(menuOpen, [menuId, route], true);
	}
	const ANIMATION_DURATION = 350;
	const toggleMobileMenu = (close) => {
		if (!mobileMenuPanel) return;
		mobileMenuPanel.style.setProperty("view-transition-name", "mobile-side-menu");
		setTimeout(() => {
			mobileMenuPanel.style.setProperty("view-transition-name", "");
		}, ANIMATION_DURATION);
		if (Core.mobileMenuOpen || close) if (document.startViewTransition) document.startViewTransition(() => {
			Core.mobileMenuOpen = 0;
		});
		else Core.mobileMenuOpen = 0;
		else if (document.startViewTransition) document.startViewTransition(() => {
			Core.mobileMenuOpen = 1;
		});
		else Core.mobileMenuOpen = 1;
	};
	user_effect(() => {
		if ($page()?.url?.pathname) {
			set(currentPathname, $page().url.pathname, true);
			if (get(module)) set(menuOpen, getMenuOpenFromRoute(get(module), get(currentPathname)), true);
		}
	});
	user_effect(() => {
		Core.toggleMobileMenu = toggleMobileMenu;
	});
	var fragment = root$1();
	var div = first_child(fragment);
	var div_1 = sibling(child(div), 2);
	each(div_1, 21, () => get(module).menus, index, ($$anchor$1, menu) => {
		const isOpen = user_derived(() => get(menuOpen)[0] === get(menu).id);
		const optionsCount = user_derived(() => get(menu).options?.length || 0);
		const menuHeight = user_derived(() => get(isOpen) ? `${get(optionsCount) * 38 + 48}px` : "48px");
		var div_2 = root_1$1();
		var button_1 = child(div_2);
		button_1.__click = () => toggleMenu(get(menu).id || 0);
		var div_3 = child(button_1);
		var span = child(div_3);
		var text = child(span, true);
		reset(span);
		var span_1 = sibling(span, 2);
		var text_1 = child(span_1, true);
		reset(span_1);
		reset(div_3);
		var node = sibling(div_3, 2);
		var consequent = ($$anchor$2) => {
			var span_2 = root_2$1();
			let classes;
			template_effect(() => classes = set_class(span_2, 1, "aKw absolute right-3 transition-all duration-300 aMB", null, classes, { "rotate-180": get(isOpen) }));
			append($$anchor$2, span_2);
		};
		if_block(node, ($$render) => {
			if (get(menu).options && get(menu).options.length > 0) $$render(consequent);
		});
		reset(button_1);
		var node_1 = sibling(button_1, 2);
		var consequent_2 = ($$anchor$2) => {
			var div_4 = root_3();
			each(div_4, 21, () => get(menu).options, index, ($$anchor$3, option) => {
				const isActive = user_derived(() => get(option).route === get(currentPathname));
				var button_2 = root_4$1();
				button_2.__click = () => navigateTo(get(option).route || "/", get(menu).id || 0);
				var div_5 = child(button_2);
				var i = child(div_5);
				var div_6 = sibling(i, 2);
				var text_2 = child(div_6, true);
				reset(div_6);
				reset(div_5);
				var div_7 = sibling(div_5, 2);
				var i_1 = child(div_7);
				var div_8 = sibling(i_1, 2);
				each(div_8, 21, () => get(option).name.split(" "), index, ($$anchor$4, word) => {
					var span_3 = root_5();
					var text_3 = child(span_3, true);
					reset(span_3);
					template_effect(() => set_text(text_3, get(word)));
					append($$anchor$4, span_3);
				});
				reset(div_8);
				reset(div_7);
				var node_2 = sibling(div_7, 2);
				var consequent_1 = ($$anchor$4) => {
					append($$anchor$4, root_6());
				};
				if_block(node_2, ($$render) => {
					if (get(isActive)) $$render(consequent_1);
				});
				reset(button_2);
				template_effect(($0) => {
					set_class(button_2, 1, `aKz w-full flex items-center px-0 py-10 relative
								hover:bg-indigo-600/20 transition-all duration-150
								border-l-2 border-transparent
								${get(isActive) ? "bg-indigo-600/30 border-indigo-400 text-white" : "text-gray-300"}`, "aMB");
					set_class(i, 1, `${(get(option).icon || "icon-box") ?? ""} mr-2`, "aMB");
					set_text(text_2, $0);
					set_class(i_1, 1, `${(get(option).icon || "icon-box") ?? ""} mr-2`, "aMB");
				}, [() => get(option).minName || get(option).name.substring(0, 2)]);
				append($$anchor$3, button_2);
			});
			reset(div_4);
			append($$anchor$2, div_4);
		};
		if_block(node_1, ($$render) => {
			if (get(menu).options && get(isOpen)) $$render(consequent_2);
		});
		reset(div_2);
		template_effect(($0, $1) => {
			set_style(div_2, `height: ${get(menuHeight) ?? ""}`);
			set_class(button_1, 1, `w-full h-48 px-12 flex items-center justify-between relative
						text-indigo-300 hover:bg-gray-800/50 transition-colors duration-400
						border-l-4 border-transparent hover:border-indigo-500
						${get(isOpen) ? "bg-gray-800 border-indigo-500" : ""}`);
			set_attribute(button_1, "aria-expanded", get(isOpen));
			set_text(text, $0);
			set_text(text_1, $1);
		}, [() => (get(menu).minName || get(menu).name.substring(0, 3)).toUpperCase(), () => get(menu).name.toUpperCase()]);
		append($$anchor$1, div_2);
	});
	reset(div_1);
	reset(div);
	var div_10 = sibling(div, 2);
	var button_3 = child(div_10);
	button_3.__click = () => toggleMobileMenu(true);
	bind_this(button_3, ($$value) => mobileMenuBackdrop = $$value, () => mobileMenuBackdrop);
	var aside = sibling(button_3, 2);
	var div_11 = child(aside);
	var button_4 = sibling(child(div_11), 2);
	button_4.__click = () => toggleMobileMenu(true);
	reset(div_11);
	var div_12 = sibling(div_11, 2);
	each(div_12, 21, () => get(module)?.menus || [], index, ($$anchor$1, menu) => {
		const isOpen = user_derived(() => get(menuOpen)[0] === get(menu).id);
		var div_13 = root_7();
		var button_5 = child(div_13);
		button_5.__click = () => toggleMenu(get(menu).id || 0);
		var span_4 = child(button_5);
		var text_4 = child(span_4, true);
		reset(span_4);
		var node_3 = sibling(span_4, 2);
		var consequent_3 = ($$anchor$2) => {
			var span_5 = root_8();
			let classes_1;
			template_effect(() => classes_1 = set_class(span_5, 1, "aKM aMB", null, classes_1, { aKN: get(isOpen) }));
			append($$anchor$2, span_5);
		};
		if_block(node_3, ($$render) => {
			if (get(menu).options && get(menu).options.length > 0) $$render(consequent_3);
		});
		reset(button_5);
		var node_4 = sibling(button_5, 2);
		var consequent_5 = ($$anchor$2) => {
			var div_14 = root_9();
			each(div_14, 21, () => get(menu).options, index, ($$anchor$3, option) => {
				const isActive = user_derived(() => get(option).route === get(currentPathname));
				var button_6 = root_10();
				button_6.__click = (e) => navigateTo(get(option).route || "/", get(menu).id || 0, e.currentTarget);
				button_6.__mousedown = (e) => e.currentTarget.classList.add("is-pressed");
				button_6.__mouseup = (e) => e.currentTarget.classList.remove("is-pressed");
				button_6.__touchstart = (e) => e.currentTarget.classList.add("is-pressed");
				button_6.__touchend = (e) => e.currentTarget.classList.remove("is-pressed");
				var node_5 = child(button_6);
				var consequent_4 = ($$anchor$4) => {
					var i_2 = root_11();
					template_effect(() => set_class(i_2, 1, `${get(option).icon ?? ""} aKS`, "aMB"));
					append($$anchor$4, i_2);
				};
				if_block(node_5, ($$render) => {
					if (get(option).icon) $$render(consequent_4);
				});
				var span_6 = sibling(node_5, 2);
				var text_5 = child(span_6, true);
				reset(span_6);
				reset(button_6);
				template_effect(() => {
					set_class(button_6, 1, `mobile-menu-option ${get(isActive) ? "is-active" : ""}`, "aMB");
					set_text(text_5, get(option).name);
				});
				event("mouseleave", button_6, (e) => e.currentTarget.classList.remove("is-pressed"));
				append($$anchor$3, button_6);
			});
			reset(div_14);
			append($$anchor$2, div_14);
		};
		if_block(node_4, ($$render) => {
			if (get(menu).options && get(isOpen)) $$render(consequent_5);
		});
		reset(div_13);
		template_effect(($0) => {
			set_class(button_5, 1, `aKK ${get(isOpen) ? "aKC" : ""}`, "aMB");
			set_text(text_4, $0);
		}, [() => get(menu).name.toUpperCase()]);
		append($$anchor$1, div_13);
	});
	reset(div_12);
	reset(aside);
	bind_this(aside, ($$value) => mobileMenuPanel = $$value, () => mobileMenuPanel);
	reset(div_10);
	template_effect(() => set_class(div_10, 1, `aKA md:hidden ${Core.mobileMenuOpen ? "aKC" : ""}`, "aMB"));
	append($$anchor, fragment);
	pop();
	$$cleanup();
}
delegate([
	"click",
	"mousedown",
	"mouseup",
	"touchstart",
	"touchend"
]);
function WorkerWrapper(options) {
	return new Worker("" + new URL("../workers/image-worker-Ddj3ULOU.js", import.meta.url).href, {
		type: "module",
		name: options?.name
	});
}
var root_1 = from_html(`<link rel="icon"/>`);
var root_4 = from_html(`<div class="p-12"><h2>Cargando...</h2></div>`);
var root_2 = from_html(`<!> <!> <!> <!>`, 1);
var root = from_html(`<!> <!>`, 1);
function _layout($$anchor, $$props) {
	push($$props, true);
	if (browser) {
		console.log("🔧 Initializing image worker...");
		try {
			Env.ImageWorkerClass = WorkerWrapper;
			Env.imageWorker = new WorkerWrapper();
			console.log("✅ Image worker initialized successfully");
		} catch (error) {
			console.error("❌ Failed to initialize image worker:", error);
		}
	}
	const redirectsToLogin = user_derived(() => {
		if (["/login"].includes(page$1.url.pathname)) return false;
		return checkIsLogin() !== 2;
	});
	onMount(() => {
		if (get(redirectsToLogin)) Env.navigate("/login");
	});
	user_effect(() => {
		Core.module = modules_default[0];
		console.log("imageWorker", Env, Env.imageWorker);
		window.addEventListener("resize", () => {
			const newDeviceType = getDeviceType();
			if (newDeviceType !== Core.deviceType) Core.deviceType = newDeviceType;
		});
		doInitServiceWorker().then(() => {
			Core.isLoading = 0;
		});
	});
	let showLayout = user_derived(() => !page$1.url.pathname.startsWith("/login"));
	var fragment = root();
	head(($$anchor$1) => {
		var link = root_1();
		$document.title = "Genix - Sistema de Gestión";
		template_effect(() => set_attribute(link, "href", favicon_default));
		append($$anchor$1, link);
	});
	var node = first_child(fragment);
	var consequent_1 = ($$anchor$1) => {
		var fragment_1 = root_2();
		var node_1 = first_child(fragment_1);
		TopLayerSelector(node_1, {});
		var node_2 = sibling(node_1, 2);
		Header(node_2, { showMenuButton: true });
		var node_3 = sibling(node_2, 2);
		SideMenu(node_3, {});
		var node_4 = sibling(node_3, 2);
		var consequent = ($$anchor$2) => {
			Page($$anchor$2, {
				title: "...",
				children: ($$anchor$3, $$slotProps) => {
					append($$anchor$3, root_4());
				},
				$$slots: { default: true }
			});
		};
		if_block(node_4, ($$render) => {
			if (Core.isLoading > 0) $$render(consequent);
		});
		append($$anchor$1, fragment_1);
	};
	if_block(node, ($$render) => {
		if (get(showLayout) && !get(redirectsToLogin)) $$render(consequent_1);
	});
	var node_5 = sibling(node, 2);
	var consequent_2 = ($$anchor$1) => {
		var fragment_3 = comment();
		snippet(first_child(fragment_3), () => $$props.children ?? noop);
		append($$anchor$1, fragment_3);
	};
	if_block(node_5, ($$render) => {
		if ((Core.isLoading === 0 || !get(showLayout)) && !get(redirectsToLogin)) $$render(consequent_2);
	});
	append($$anchor, fragment);
	pop();
}
export { _layout as component, _layout_exports as universal };
