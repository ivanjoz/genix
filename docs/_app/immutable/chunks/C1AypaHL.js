import { $ as proxy, A as if_block, B as tick, C as clsx, E as snippet, G as user_effect, L as delegate, M as append, P as from_html, Q as sibling, S as set_class, W as template_effect, X as child, Z as first_child, _t as reset, a as prop, at as state, f as bind_this, j as set_text, lt as pop, rt as set, st as user_derived, ut as push, v as set_attribute, x as set_style, z as get } from "./CTB4HzdN.js";
import { t as Core, y as Env } from "./BwrZ3UQO.js";
import { t as OptionsStrip } from "./CGFlWCTf.js";
var root_2 = from_html(`<button class="bx-red mr-10 lh-10" aria-label="Eliminar"><i class="icon-trash"></i></button>`);
var root_3 = from_html(`<button class="bx-blue mr-10 lh-10" aria-label="Guardar"><i></i> <span class="_11 aMq"> </span></button>`);
var root_1 = from_html(`<div><div class="flex items-center justify-between"><div> </div> <div class="shrink-0 flex items-center mb-2"><!> <!> <button class="bx-yellow" title="close"><i class="icon-cancel"></i></button></div></div> <!> <div><!></div></div>`);
var root_5 = from_html(`<div class="w-page"><!></div>`);
var root = from_html(`<!> <!>`, 1);
function Layer($$anchor, $$props) {
	push($$props, true);
	let divLayer;
	let previousShowSideLayer = state(proxy(Core.showSideLayer));
	let isInTransition = state(false);
	let selected = prop($$props, "selected", 15);
	let isVisible = state(proxy(Core.deviceType !== 3 || $$props.type !== "side"));
	const layerWidth = user_derived(() => {
		return Env.sideLayerSize ? Env.sideLayerSize + "px" : "";
	});
	const contentWidth = user_derived(() => {
		return Core.showSideLayer > 0 && Core.deviceType !== 3 ? `calc(var(--page-width) - ${get(layerWidth) || "0"} - 8px)` : void 0;
	});
	const closeLayer = () => {
		Core.openSideLayer(0);
	};
	user_effect(() => {
		if ($$props.type !== "side" || Core.deviceType !== 3) {
			set(isVisible, Core.showSideLayer === $$props.id);
			set(previousShowSideLayer, Core.showSideLayer, true);
			return;
		}
		if (get(previousShowSideLayer) !== Core.showSideLayer) {
			const isOpening = Core.showSideLayer === $$props.id;
			if (isOpening && typeof document.startViewTransition === "function") {
				if (get(previousShowSideLayer) !== 0) {
					const oldLayer = document.querySelector(`[data-layer-id="${get(previousShowSideLayer)}"]`);
					if (oldLayer) oldLayer.style.setProperty("view-transition-name", "");
				}
				document.startViewTransition(async () => {
					set(isInTransition, true);
					set(isVisible, true);
					await tick();
				}).finished.finally(() => {
					set(isInTransition, false);
				});
			} else if (isOpening) set(isVisible, true);
			else if (get(previousShowSideLayer) === $$props.id) set(isVisible, false);
			set(previousShowSideLayer, Core.showSideLayer, true);
		}
	});
	user_effect(() => {
		if (Core.showSideLayer || $$props.type) console.log("updated 122:", Core.showSideLayer, $$props.type, "| open:", Core.showSideLayer === $$props.id);
	});
	var fragment = root();
	var node = first_child(fragment);
	var consequent_3 = ($$anchor$1) => {
		var div = root_1();
		let classes;
		var div_1 = child(div);
		var div_2 = child(div_1);
		var text = child(div_2, true);
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		var node_1 = child(div_3);
		var consequent = ($$anchor$2) => {
			var button = root_2();
			button.__click = function(...$$args) {
				$$props.onDelete?.apply(this, $$args);
			};
			append($$anchor$2, button);
		};
		if_block(node_1, ($$render) => {
			if ($$props.onDelete) $$render(consequent);
		});
		var node_2 = sibling(node_1, 2);
		var consequent_1 = ($$anchor$2) => {
			var button_1 = root_3();
			button_1.__click = function(...$$args) {
				$$props.onSave?.apply(this, $$args);
			};
			var i = child(button_1);
			var span = sibling(i, 2);
			var text_1 = child(span, true);
			reset(span);
			reset(button_1);
			template_effect(() => {
				set_class(i, 1, clsx($$props.saveButtonIcon || "icon-floppy"), "aMq");
				set_text(text_1, $$props.saveButtonName || "Guardar");
			});
			append($$anchor$2, button_1);
		};
		if_block(node_2, ($$render) => {
			if ($$props.onSave) $$render(consequent_1);
		});
		var button_2 = sibling(node_2, 2);
		button_2.__click = (ev) => {
			ev.stopPropagation();
			closeLayer();
			if ($$props.onClose) if (Core.deviceType === 3) setTimeout(() => {
				$$props.onClose();
			}, 300);
			else $$props.onClose();
		};
		reset(div_3);
		reset(div_1);
		var node_3 = sibling(div_1, 2);
		var consequent_2 = ($$anchor$2) => {
			OptionsStrip($$anchor$2, {
				get options() {
					return $$props.options;
				},
				get selected() {
					return selected();
				},
				useMobileGrid: true,
				onSelect: (e) => {
					selected(e[0]);
				},
				css: "mt-2"
			});
		};
		if_block(node_3, ($$render) => {
			if (($$props.options || []).length > 0) $$render(consequent_2);
		});
		var div_4 = sibling(node_3, 2);
		snippet(child(div_4), () => $$props.children);
		reset(div_4);
		reset(div);
		bind_this(div, ($$value) => divLayer = $$value, () => divLayer);
		template_effect(() => {
			classes = set_class(div, 1, `flex flex-col w-800 ${($$props.css || "") ?? ""}`, "aMq", classes, {
				_8: $$props.contentOverflow,
				_1: $$props.type === "side",
				_2: $$props.type === "bottom",
				aJP: get(isInTransition),
				visible: get(isVisible) || Core.deviceType !== 3 || $$props.type !== "side"
			});
			set_attribute(div, "data-layer-id", $$props.id);
			set_style(div, get(layerWidth) ? `width: ${get(layerWidth)};` : "");
			set_class(div_2, 1, `overflow-hidden text-nowrap mr-8 ${$$props.titleCss ?? ""}`, "aMq");
			set_text(text, $$props.title);
			set_class(div_4, 1, `_4 grow-1 ${$$props.contentCss ?? ""}`, "aMq");
		});
		append($$anchor$1, div);
	};
	if_block(node, ($$render) => {
		if (Core.showSideLayer === $$props.id && ($$props.type === "side" || $$props.type === "bottom")) $$render(consequent_3);
	});
	var node_5 = sibling(node, 2);
	var consequent_4 = ($$anchor$1) => {
		var div_5 = root_5();
		let styles;
		snippet(child(div_5), () => $$props.children);
		reset(div_5);
		template_effect(() => styles = set_style(div_5, "", styles, { width: get(contentWidth) }));
		append($$anchor$1, div_5);
	};
	if_block(node_5, ($$render) => {
		if ($$props.type == "content") $$render(consequent_4);
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click"]);
export { Layer as t };
