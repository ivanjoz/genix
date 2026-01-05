import { $ as proxy, A as if_block, C as clsx, D as html, G as user_effect, L as delegate, M as append, N as comment, P as from_html, Q as sibling, R as event, S as set_class, V as untrack, W as template_effect, X as child, Z as first_child, _ as remove_input_defaults, _t as reset, a as prop, at as state, j as set_text, lt as pop, m as bind_value, mt as snapshot, q as remove_textarea_child, rt as set, st as user_derived, ut as push, v as set_attribute, z as get } from "./CTB4HzdN.js";
import { t as components_module_default } from "./cyfMRGwx.js";
var root_1 = from_html(`<div><div></div></div> <div> <!></div> <div><div></div></div> <div><div></div></div>`, 1);
var root_2 = from_html(`<div><div></div></div>`);
var root_3 = from_html(`<textarea></textarea>`);
var root_4 = from_html(`<input/>`);
var root_6 = from_html(`<div> </div>`);
var root = from_html(`<div><!> <div><!> <!> <!> <!></div></div>`);
function Input($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 15);
	const baseDecimalsValue = $$props.baseDecimals ? 10 ** $$props.baseDecimals : 0;
	const checkIfInputIsValid = () => {
		if (!$$props.required || $$props.disabled) return 0;
		if (!saveOn() || !$$props.save) return 1;
		const value = saveOn()[$$props.save];
		let pass = !$$props.required;
		if ($$props.validator) pass = $$props.validator(value);
		else if (value || value === 0) pass = true;
		return pass ? 2 : 1;
	};
	let inputValue = state("");
	let isInputValid = state(proxy(checkIfInputIsValid()));
	let isChange = 0;
	let focusValue = null;
	const onKeyUp = (ev, isBlur) => {
		ev.stopPropagation();
		let value = ev.target.value;
		if ($$props.type === "number") {
			if (!isBlur && !value && ev.key === "-") return;
			if (isNaN(value)) value = void 0;
			else value = parseFloat(value);
		}
		if (isBlur && $$props.validator && !$$props.validator(value)) {
			set(inputValue, focusValue, true);
			if (saveOn() && $$props.save) {
				if (baseDecimalsValue && typeof get(inputValue) === "number") set(inputValue, Math.round(get(inputValue) * baseDecimalsValue), true);
				saveOn(saveOn()[$$props.save] = get(inputValue), true);
			}
			return;
		}
		if ($$props.transform && isBlur) value = $$props.transform(value);
		untrack(() => {
			if (saveOn() && $$props.save) {
				let valueSaved = value;
				if (baseDecimalsValue && typeof valueSaved === "number") valueSaved = Math.round(valueSaved * baseDecimalsValue);
				saveOn(saveOn()[$$props.save] = valueSaved, true);
				set(isInputValid, checkIfInputIsValid(), true);
			}
		});
		if (!isBlur) isChange = 1;
		set(inputValue, value, true);
	};
	const iconValid = () => {
		if (!get(isInputValid)) return null;
		else if (get(isInputValid) === 2) return `<i class="v-icon icon-ok c-green"></i>`;
		else if (get(isInputValid) === 1) return `<i class="v-icon icon-attention c-red"></i>`;
		return null;
	};
	let lastSaveOn;
	const doSave = () => {
		untrack(() => {
			const v = saveOn()[$$props.save];
			set(inputValue, typeof v === "number" ? v : v || "", true);
			if (baseDecimalsValue && typeof get(inputValue) === "number") set(inputValue, get(inputValue) / baseDecimalsValue);
			set(isInputValid, checkIfInputIsValid(), true);
		});
	};
	user_effect(() => {
		if (!saveOn() || !$$props.save) return;
		if (lastSaveOn === saveOn()) return;
		lastSaveOn = saveOn();
		if (saveOn()[$$props.save] !== get(inputValue)) doSave();
	});
	user_effect(() => {
		if ($$props.dependencyValue) doSave();
	});
	let cN = user_derived(() => `${components_module_default.input} p-rel${$$props.css ? ` ${$$props.css}` : ""}${!$$props.label ? " no-label" : ""}`);
	var div = root();
	var node = child(div);
	var consequent = ($$anchor$1) => {
		var fragment = root_1();
		var div_1 = first_child(fragment);
		var div_2 = sibling(div_1, 2);
		var text = child(div_2, true);
		html(sibling(text), () => iconValid() || "");
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		var div_4 = sibling(div_3, 2);
		template_effect(() => {
			set_class(div_1, 1, clsx(components_module_default.input_lab_cell_left));
			set_class(div_2, 1, clsx(components_module_default.input_lab));
			set_text(text, $$props.label);
			set_class(div_3, 1, clsx(components_module_default.input_lab_cell_right));
			set_class(div_4, 1, clsx(components_module_default.input_shadow_layer));
		});
		append($$anchor$1, fragment);
	};
	if_block(node, ($$render) => {
		if ($$props.label) $$render(consequent);
	});
	var div_5 = sibling(node, 2);
	var node_2 = child(div_5);
	var consequent_1 = ($$anchor$1) => {
		var div_6 = root_2();
		template_effect(() => set_class(div_6, 1, clsx(components_module_default.input_div_1)));
		append($$anchor$1, div_6);
	};
	if_block(node_2, ($$render) => {
		if ($$props.label) $$render(consequent_1);
	});
	var node_3 = sibling(node_2, 2);
	var consequent_2 = ($$anchor$1) => {
		var textarea = root_3();
		remove_textarea_child(textarea);
		textarea.__keyup = (ev) => {
			onKeyUp(ev);
		};
		template_effect(() => {
			set_class(textarea, 1, `w-full ${components_module_default.input_inp} ${$$props.inputCss || ""}`);
			set_attribute(textarea, "placeholder", $$props.placeholder || "");
			textarea.disabled = $$props.disabled;
			set_attribute(textarea, "rows", $$props.rows);
		});
		event("blur", textarea, (ev) => {
			console.log("input saveon:", snapshot(saveOn()));
			onKeyUp(ev, true);
			if ($$props.onChange && isChange) {
				$$props.onChange();
				isChange = 0;
			}
		});
		bind_value(textarea, () => get(inputValue), ($$value) => set(inputValue, $$value));
		append($$anchor$1, textarea);
	};
	var alternate = ($$anchor$1) => {
		var input = root_4();
		remove_input_defaults(input);
		input.__keyup = (ev) => {
			onKeyUp(ev);
		};
		template_effect(() => {
			set_class(input, 1, `w-full ${components_module_default.input_inp ?? ""} ${($$props.inputCss || "") ?? ""}`);
			set_attribute(input, "type", $$props.type || "search");
			set_attribute(input, "placeholder", $$props.placeholder || "");
			input.disabled = $$props.disabled;
		});
		event("focus", input, (ev) => {
			focusValue = ev.target.value;
		});
		event("blur", input, (ev) => {
			onKeyUp(ev, true);
			if ($$props.onChange && isChange) {
				$$props.onChange();
				isChange = 0;
			}
			focusValue = null;
		});
		bind_value(input, () => get(inputValue), ($$value) => set(inputValue, $$value));
		append($$anchor$1, input);
	};
	if_block(node_3, ($$render) => {
		if ($$props.useTextArea) $$render(consequent_2);
		else $$render(alternate, false);
	});
	var node_4 = sibling(node_3, 2);
	var consequent_3 = ($$anchor$1) => {
		var fragment_1 = comment();
		html(first_child(fragment_1), () => iconValid() || "");
		append($$anchor$1, fragment_1);
	};
	if_block(node_4, ($$render) => {
		if (!$$props.label) $$render(consequent_3);
	});
	var node_6 = sibling(node_4, 2);
	var consequent_4 = ($$anchor$1) => {
		var div_7 = root_6();
		var text_1 = child(div_7, true);
		reset(div_7);
		template_effect(() => {
			set_class(div_7, 1, components_module_default.input_post_value);
			set_text(text_1, $$props.postValue);
		});
		append($$anchor$1, div_7);
	};
	if_block(node_6, ($$render) => {
		if ($$props.postValue) $$render(consequent_4);
	});
	reset(div_5);
	reset(div);
	template_effect(() => {
		set_class(div, 1, clsx(get(cN)));
		set_class(div_5, 1, `${components_module_default.input_div} flex w-full`);
	});
	append($$anchor, div);
	pop();
}
delegate(["keyup"]);
export { Input as t };
