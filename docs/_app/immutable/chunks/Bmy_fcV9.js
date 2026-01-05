import { $ as proxy, A as if_block, G as user_effect, L as delegate, M as append, O as each, P as from_html, Q as sibling, S as set_class, V as untrack, W as template_effect, X as child, _t as reset, a as prop, at as state, j as set_text, k as index, lt as pop, rt as set, st as user_derived, ut as push, v as set_attribute, z as get } from "./CTB4HzdN.js";
var root_2 = from_html(`<i class="icon-ok"></i>`);
var root_1 = from_html(`<div class="flex items-center mr-10"><button><!></button> <label> </label></div>`);
var root = from_html(`<div></div>`);
function CheckboxOptions($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 15);
	let optionsSelected = state(proxy([]));
	const onSelect = (e) => {
		const id = e[$$props.keyId];
		if ($$props.type == "multiple") if (get(optionsSelected).includes(id)) set(optionsSelected, get(optionsSelected).filter((x) => x !== id), true);
		else get(optionsSelected).push(id);
		else if (get(optionsSelected).includes(id)) set(optionsSelected, [], true);
		else set(optionsSelected, [id], true);
		if (saveOn() && $$props.save) if ($$props.type === "multiple") saveOn(saveOn()[$$props.save] = get(optionsSelected), true);
		else saveOn(saveOn()[$$props.save] = get(optionsSelected)[0] || void 0, true);
	};
	let lastSaveOn;
	user_effect(() => {
		if (!saveOn() || !$$props.save) return;
		if (lastSaveOn === saveOn()) return;
		lastSaveOn = saveOn();
		untrack(() => {
			if ($$props.type === "multiple") set(optionsSelected, saveOn()[$$props.save] || [], true);
			else set(optionsSelected, [saveOn()[$$props.save] || []], true);
		});
	});
	var div = root();
	each(div, 21, () => $$props.options, index, ($$anchor$1, opt) => {
		const isSelected = user_derived(() => get(optionsSelected).includes(get(opt)[$$props.keyId]));
		var div_1 = root_1();
		var button = child(div_1);
		let classes;
		button.__click = (ev) => {
			ev.stopPropagation();
			onSelect(get(opt));
		};
		var node = child(button);
		var consequent = ($$anchor$2) => {
			append($$anchor$2, root_2());
		};
		if_block(node, ($$render) => {
			if (get(isSelected)) $$render(consequent);
		});
		reset(button);
		var label = sibling(button, 2);
		var text = child(label, true);
		reset(label);
		reset(div_1);
		template_effect(() => {
			classes = set_class(button, 1, "flex mr-4 pt-1 items-center p-0 lh-10 justify-center rounded-[4px] shrink-0 w-28 h-26 _1 aMo", null, classes, { _2: get(isSelected) });
			set_attribute(button, "aria-label", get(opt)[$$props.keyName]);
			set_text(text, get(opt)[$$props.keyName]);
		});
		append($$anchor$1, div_1);
	});
	reset(div);
	template_effect(() => set_class(div, 1, `flex ${$$props.css ?? ""}`, "aMo"));
	append($$anchor, div);
	pop();
}
delegate(["click"]);
export { CheckboxOptions as t };
