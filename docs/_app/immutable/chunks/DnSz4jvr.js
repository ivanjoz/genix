import { $ as proxy, A as if_block, B as tick, C as clsx, E as snippet, G as user_effect, M as append, N as comment, P as from_html, Q as sibling, S as set_class, W as template_effect, X as child, Z as first_child, _t as reset, a as prop, at as state, f as bind_this, lt as pop, rt as set, st as user_derived, ut as push, v as set_attribute, vt as noop, x as set_style, z as get } from "./CTB4HzdN.js";
import { i as parseSVG } from "./DwKmqXYU.js";
import { t as angle_default } from "./B0DNpKVT.js";
var root = from_html(`<div class="_1 absolute top-0 aMF"><!></div>`);
function Portal($$anchor, $$props) {
	push($$props, true);
	let target = prop($$props, "target", 3, void 0);
	let contentElement = state(null);
	let portalTarget = state(null);
	user_effect(() => {
		if (!get(contentElement)) return;
		const targetElement = target() || document.body;
		set(portalTarget, targetElement, true);
		targetElement.appendChild(get(contentElement));
		return () => {
			if (get(contentElement) && get(contentElement).parentNode) get(contentElement).parentNode.removeChild(get(contentElement));
		};
	});
	var div = root();
	let styles;
	snippet(child(div), () => $$props.children ?? noop);
	reset(div);
	bind_this(div, ($$value) => set(contentElement, $$value), () => get(contentElement));
	template_effect(() => styles = set_style(div, "", styles, { "z-index": $$props.zIndex ? $$props.zIndex : void 0 }));
	append($$anchor, div);
	pop();
}
function calculatePosition(referenceElement, floatingElement, options = {}) {
	const { offset = 8, preferredPlacement = "bottom", fitViewport = true } = options;
	const refRect = referenceElement.getBoundingClientRect();
	const floatRect = floatingElement.getBoundingClientRect();
	const viewportWidth = window.innerWidth;
	const viewportHeight = window.innerHeight;
	const scrollX = window.scrollX || window.pageXOffset;
	const scrollY = window.scrollY || window.pageYOffset;
	const spaces = {
		top: refRect.top,
		bottom: viewportHeight - refRect.bottom,
		left: refRect.left,
		right: viewportWidth - refRect.right
	};
	const fits = {
		top: spaces.top >= floatRect.height + offset,
		bottom: spaces.bottom >= floatRect.height + offset,
		left: spaces.left >= floatRect.width + offset,
		right: spaces.right >= floatRect.width + offset
	};
	const basePlacement = preferredPlacement.split("-")[0];
	let finalPlacement = preferredPlacement;
	let baseFinalPlacement = basePlacement;
	if (!fits[basePlacement]) {
		if (basePlacement === "bottom" || basePlacement === "top") if (basePlacement === "bottom" && fits.top) baseFinalPlacement = "top";
		else if (basePlacement === "top" && fits.bottom) baseFinalPlacement = "bottom";
		else if (fits.right) baseFinalPlacement = "right";
		else if (fits.left) baseFinalPlacement = "left";
		else baseFinalPlacement = spaces.bottom >= spaces.top ? "bottom" : "top";
		else if (basePlacement === "right" && fits.left) baseFinalPlacement = "left";
		else if (basePlacement === "left" && fits.right) baseFinalPlacement = "right";
		else if (fits.bottom) baseFinalPlacement = "bottom";
		else if (fits.top) baseFinalPlacement = "top";
		else baseFinalPlacement = spaces.right >= spaces.left ? "right" : "left";
		finalPlacement = baseFinalPlacement;
	}
	let top = 0;
	let left = 0;
	switch (baseFinalPlacement) {
		case "bottom":
			top = refRect.bottom + offset;
			left = refRect.left + refRect.width / 2 - floatRect.width / 2;
			break;
		case "top":
			top = refRect.top - floatRect.height - offset;
			left = refRect.left + refRect.width / 2 - floatRect.width / 2;
			break;
		case "right":
			top = refRect.top + refRect.height / 2 - floatRect.height / 2;
			left = refRect.right + offset;
			break;
		case "left":
			top = refRect.top + refRect.height / 2 - floatRect.height / 2;
			left = refRect.left - floatRect.width - offset;
			break;
	}
	if (preferredPlacement.includes("-")) {
		const alignment = preferredPlacement.split("-")[1];
		if (baseFinalPlacement === "top" || baseFinalPlacement === "bottom") {
			if (alignment === "start") left = refRect.left;
			else if (alignment === "end") left = refRect.right - floatRect.width;
		} else if (alignment === "start") top = refRect.top;
		else if (alignment === "end") top = refRect.bottom - floatRect.height;
	}
	top += scrollY;
	left += scrollX;
	if (fitViewport) {
		const minLeft = offset + scrollX;
		const maxLeft = viewportWidth - floatRect.width - offset + scrollX;
		const minTop = offset + scrollY;
		const maxTop = viewportHeight - floatRect.height - offset + scrollY;
		left = Math.max(minLeft, Math.min(left, maxLeft));
		top = Math.max(minTop, Math.min(top, maxTop));
	}
	const maxHeight = viewportHeight - offset * 2;
	const maxWidth = viewportWidth - offset * 2;
	return {
		top: Math.round(top),
		left: Math.round(left),
		placement: finalPlacement,
		maxHeight,
		maxWidth
	};
}
var root_2 = from_html(`<div><!> <div><img alt=""/></div></div>`);
function Popover2($$anchor, $$props) {
	push($$props, true);
	let open = prop($$props, "open", 3, false), placement = prop($$props, "placement", 3, "bottom"), offset = prop($$props, "offset", 3, 8), fitViewport = prop($$props, "fitViewport", 3, true), className = prop($$props, "class", 3, ""), customStyle = prop($$props, "style", 3, "");
	let floatingElement = state(null);
	let position = state(proxy({
		top: 0,
		left: 0,
		placement: placement()
	}));
	user_effect(() => {
		if (open() && $$props.referenceElement && get(floatingElement)) {
			console.log("referenceElement", $$props.referenceElement);
			updatePosition();
		}
	});
	user_effect(() => {
		if (!open()) return;
		const handleUpdate = () => {
			if (open() && $$props.referenceElement && get(floatingElement)) updatePosition();
		};
		window.addEventListener("scroll", handleUpdate, true);
		window.addEventListener("resize", handleUpdate);
		return () => {
			window.removeEventListener("scroll", handleUpdate, true);
			window.removeEventListener("resize", handleUpdate);
		};
	});
	async function updatePosition() {
		if (!$$props.referenceElement || !get(floatingElement)) return;
		await tick();
		const result = calculatePosition($$props.referenceElement, get(floatingElement), {
			offset: offset(),
			preferredPlacement: placement(),
			fitViewport: fitViewport()
		});
		set(position, result, true);
		if ($$props.onPositionUpdate) $$props.onPositionUpdate(result);
	}
	const computedStyle = user_derived(() => () => {
		if (!open()) return "display: none;";
		return `
			position: absolute;
			top: ${get(position).top}px;
			left: ${get(position).left}px;
			z-index: 210;
			${customStyle()}
		`.trim();
	});
	var fragment = comment();
	var node = first_child(fragment);
	var consequent = ($$anchor$1) => {
		Portal($$anchor$1, {
			children: ($$anchor$2, $$slotProps) => {
				var div = root_2();
				var node_1 = child(div);
				snippet(node_1, () => $$props.children ?? noop);
				var div_1 = sibling(node_1, 2);
				var img = child(div_1);
				set_class(img, 1, `_5 w-24 h-24 top`);
				reset(div_1);
				reset(div);
				bind_this(div, ($$value) => set(floatingElement, $$value), () => get(floatingElement));
				template_effect(($0, $1) => {
					set_class(div, 1, `${className() ?? ""} ps-${get(position).placement ?? ""}`);
					set_style(div, $0);
					set_class(div_1, 1, clsx([
						"absolute overflow-hidden h-18 flex justify-center w-full",
						get(position).placement === "bottom" && "top-[-18px]",
						get(position).placement === "top" && "bottom-[-18px] rotate-180"
					]));
					set_attribute(img, "src", $1);
				}, [() => get(computedStyle)(), () => parseSVG(angle_default)]);
				append($$anchor$2, div);
			},
			$$slots: { default: true }
		});
	};
	if_block(node, ($$render) => {
		if (open()) $$render(consequent);
	});
	append($$anchor, fragment);
	pop();
}
export { Portal as n, Popover2 as t };
