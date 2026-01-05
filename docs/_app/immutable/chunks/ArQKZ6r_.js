import { A as if_block, C as clsx, G as user_effect, L as delegate, M as append, N as comment, O as each, P as from_html, Q as sibling, R as event, S as set_class, V as untrack, W as template_effect, X as child, Z as first_child, _ as remove_input_defaults, _t as reset, a as prop, at as state, f as bind_this, j as set_text, k as index, lt as pop, rt as set, st as user_derived, ut as push, v as set_attribute, y as set_value, z as get } from "./CTB4HzdN.js";
import { t as components_module_default } from "./cyfMRGwx.js";
var root_3 = from_html(`<div class="aJr text-center flex items-center justify-center text-[13px] ff-bold aMy"> </div>`);
var root_6 = from_html(`<div class="aJw aMy"></div>`);
var root_5 = from_html(`<button type="button"> <!></button>`);
var root_4 = from_html(`<div class="flex"><div class="aJq text-[13px] ff-bold text-center flex items-center justify-center c-purple aMy"> </div> <!></div>`);
var root_2 = from_html(`<div class="aJp aMy" role="presentation"><div class="flex justify-between items-center mb-[2px]"><button class="h2 aJx flex items-center justify-center p-0 bg-transparent border-0 aMy" type="button">«</button> <button class="h2 aJx flex items-center justify-center p-0 bg-transparent border-0 aMy" type="button">‹</button> <div class="aJy flex items-center justify-center aMy"><div class="mr-[4px]"> </div> <div> </div></div> <button class="h2 aJx flex items-center justify-center p-0 bg-transparent border-0 aMy" type="button">›</button> <button class="h2 aJx flex items-center justify-center p-0 bg-transparent border-0 aMy" type="button">»</button></div> <div class="flex"><div class="aJq base text-[13px] ff-bold c-purple aMy"></div> <!></div> <!></div>`);
var root_1 = from_html(`<div><div><div></div></div> <div> </div> <div><div></div></div> <div><div></div></div> <div><div><div></div></div> <input type="text"/></div> <!></div>`);
var root_9 = from_html(`<div class="aJr text-center flex items-center justify-center text-[13px] ff-bold aMy"> </div>`);
var root_12 = from_html(`<div class="aJw aMy"></div>`);
var root_11 = from_html(`<button type="button"> <!></button>`);
var root_10 = from_html(`<div class="flex"><div class="aJq text-[13px] ff-bold text-center flex items-center justify-center c-purple aMy"> </div> <!></div>`);
var root_8 = from_html(`<div class="aJp aMy" role="presentation"><div class="flex justify-between items-center mb-[2px]"><button class="h2 aJx flex items-center justify-center p-0 bg-transparent border-0 aMy" type="button">«</button> <button class="h2 aJx flex items-center justify-center p-0 bg-transparent border-0 aMy" type="button">‹</button> <div class="aJy flex items-center justify-center aMy"><div class="mr-[4px]"> </div> <div> </div></div> <button class="h2 aJx flex items-center justify-center p-0 bg-transparent border-0 aMy" type="button">›</button> <button class="h2 aJx flex items-center justify-center p-0 bg-transparent border-0 aMy" type="button">»</button></div> <div class="flex"><div class="aJq base text-[13px] ff-bold c-purple aMy"></div> <!></div> <!></div>`);
var root_7 = from_html(`<div><div><div></div></div> <div><div><div></div></div> <input type="text"/></div> <!></div>`);
function DateInput($$anchor, $$props) {
	push($$props, true);
	let saveOn = prop($$props, "saveOn", 15), label = prop($$props, "label", 3, ""), css = prop($$props, "css", 3, ""), inputCss = prop($$props, "inputCss", 3, ""), placeholder = prop($$props, "placeholder", 3, "DD-MM-YYYY");
	prop($$props, "required", 3, false);
	let disabled = prop($$props, "disabled", 3, false);
	prop($$props, "type", 3, "unix");
	const weekDaysNames = [
		{
			n: 1,
			name: "LU"
		},
		{
			n: 2,
			name: "MA"
		},
		{
			n: 3,
			name: "MI"
		},
		{
			n: 4,
			name: "JU"
		},
		{
			n: 5,
			name: "VI"
		},
		{
			n: 6,
			name: "SA"
		},
		{
			n: 7,
			name: "DO"
		}
	];
	const monthsNamesMap = new Map([
		{
			n: 1,
			name: "Enero"
		},
		{
			n: 2,
			name: "Febrero"
		},
		{
			n: 3,
			name: "Marzo"
		},
		{
			n: 4,
			name: "Abril"
		},
		{
			n: 5,
			name: "Mayo"
		},
		{
			n: 6,
			name: "Junio"
		},
		{
			n: 7,
			name: "Julio"
		},
		{
			n: 8,
			name: "Agosto"
		},
		{
			n: 9,
			name: "Septiembre"
		},
		{
			n: 10,
			name: "Octubre"
		},
		{
			n: 11,
			name: "Noviembre"
		},
		{
			n: 12,
			name: "Diciembre"
		}
	].map((m) => [m.n, m]));
	const fechaToday = /* @__PURE__ */ new Date();
	const offset = fechaToday.getTimezoneOffset() * 60;
	const fechaTodayUnix = Math.floor((fechaToday.getTime() - offset * 1e3) / 864e5);
	const month_ = fechaToday.getFullYear() * 100 + (fechaToday.getMonth() + 1);
	let monthSelected = state(month_);
	let fechaSelected = state(0);
	let fechaFocus = state(0);
	let showCalendar = state(false);
	let inputValue = state("");
	let avoidCloseOnBlur = false;
	let inputElement = state(void 0);
	const parseMonth = (yearMonth) => {
		const yearMonthString = String(yearMonth);
		const year = parseInt(yearMonthString.substring(0, 4));
		const month = parseInt(yearMonthString.substring(4, 6));
		return {
			name: monthsNamesMap.get(month)?.name || "-",
			year,
			month
		};
	};
	const startOfISOWeek = (date) => {
		const d = new Date(date);
		const day = d.getDay();
		const diff = (day === 0 ? -6 : 1) - day;
		d.setDate(d.getDate() + diff);
		d.setHours(0, 0, 0, 0);
		return d;
	};
	const getISOWeek = (date) => {
		const d = new Date(date);
		d.setHours(0, 0, 0, 0);
		d.setDate(d.getDate() + 4 - (d.getDay() || 7));
		const yearStart = new Date(d.getFullYear(), 0, 1);
		return Math.ceil(((d.getTime() - yearStart.getTime()) / 864e5 + 1) / 7);
	};
	const getISOWeekYear = (date) => {
		const d = new Date(date);
		d.setDate(d.getDate() + 4 - (d.getDay() || 7));
		return d.getFullYear();
	};
	const semanasDias = user_derived(() => {
		let fecha;
		if (get(monthSelected)) {
			const { year, month } = parseMonth(get(monthSelected));
			fecha = new Date(year, month - 1, 1, 0, 0, 0);
		} else fecha = /* @__PURE__ */ new Date();
		const monthStart = new Date(fecha.getFullYear(), fecha.getMonth(), 1);
		const monthEnd = new Date(fecha.getFullYear(), fecha.getMonth() + 1, 0);
		const monthCurrent = fecha.getMonth();
		let fechaStart = startOfISOWeek(monthStart);
		const semanas = [];
		while (fechaStart.getTime() <= monthEnd.getTime()) {
			const fechaStarTime = fechaStart.getTime();
			const fechaStartUnix = Math.floor((fechaStarTime - offset * 1e3) / 864e5);
			const year = getISOWeekYear(fechaStart);
			const week = getISOWeek(fechaStart);
			const weekDays = [];
			for (let i = 0; i < 7; i++) {
				const date = new Date(fechaStarTime + 864e5 * i);
				const fechaUnix = fechaStartUnix + i;
				const day = date.getDate();
				const month = date.getFullYear() * 100 + (date.getMonth() + 1);
				weekDays.push({
					date,
					fechaUnix,
					day,
					month
				});
			}
			semanas.push({
				year,
				week,
				weekDays,
				monthCurrent
			});
			fechaStart = new Date(fechaStarTime + 864e5 * 7);
		}
		return semanas;
	});
	const monthName = user_derived(() => parseMonth(get(monthSelected)));
	const changeMonth = (count) => {
		const mn = get(monthName);
		const fecha = new Date(mn.year, mn.month - 1, 1, 0, 0, 0);
		fecha.setMonth(fecha.getMonth() + count);
		set(monthSelected, fecha.getFullYear() * 100 + (fecha.getMonth() + 1));
		get(inputElement)?.focus();
	};
	const makeFechaFormat = (fechaUnix) => {
		if (!fechaUnix) return "";
		const fechaHoraUnix = fechaUnix * 86400 + offset;
		const date = /* @__PURE__ */ new Date(fechaHoraUnix * 1e3);
		return `${String(date.getDate()).padStart(2, "0")}-${String(date.getMonth() + 1).padStart(2, "0")}-${date.getFullYear()}`;
	};
	const parseValue = (value) => {
		value = value.trim().substring(0, 10).replaceAll("/", "-");
		const todayYear = String(fechaToday.getFullYear());
		const arrValue = value.split("-").filter((x) => x);
		let day = isNaN(arrValue[0]) ? 0 : parseInt(arrValue[0]);
		let month = isNaN(arrValue[1]) ? 0 : parseInt(arrValue[1]);
		let yearString = arrValue[2] || "";
		if (yearString.length !== 4) {
			for (let i = 0; i < todayYear.length; i++) if (!yearString[i]) yearString += todayYear[i];
		}
		let year = isNaN(yearString) ? 0 : parseInt(yearString);
		const isCompleted = day && month && yearString.length === 4;
		let fechaUnixAutocomplated = 0;
		let fechaAutocomplated = null;
		if (day) {
			if (!month) month = fechaToday.getMonth() + 1;
			if (!year) year = fechaToday.getFullYear();
			fechaAutocomplated = new Date(year, month - 1, day, 0, 0, 0);
			if (fechaAutocomplated.getTime) fechaUnixAutocomplated = Math.floor((fechaAutocomplated.getTime() - offset * 1e3) / 864e5);
			else fechaAutocomplated = null;
		}
		return {
			isCompleted,
			fechaUnixAutocomplated,
			fechaAutocomplated,
			day,
			month,
			year
		};
	};
	const setAutocompletedValue = (value) => {
		const acv = parseValue(value);
		if (acv.fechaAutocomplated) {
			set(monthSelected, acv.fechaAutocomplated.getFullYear() * 100 + (acv.fechaAutocomplated.getMonth() + 1));
			set(fechaFocus, acv.fechaUnixAutocomplated, true);
		} else set(fechaFocus, 0);
	};
	const changeFechaSelected = (fechaUnix) => {
		untrack(() => {
			if ($$props.save && saveOn()) saveOn(saveOn()[$$props.save] = fechaUnix, true);
		});
		set(fechaSelected, fechaUnix, true);
	};
	const regexKeys = new Set([
		"1",
		"2",
		"3",
		"4",
		"5",
		"6",
		"7",
		"8",
		"9",
		"0",
		"-",
		"/"
	]);
	const regexKeysPress = new Set([
		...regexKeys,
		"Backspace",
		"Control",
		"c",
		"v",
		"x",
		"Tab"
	]);
	let timeoutId;
	const throttle = (fn, delay) => {
		if (timeoutId) clearTimeout(timeoutId);
		timeoutId = setTimeout(fn, delay);
	};
	const handleKeyDown = (ev) => {
		if (!regexKeysPress.has(ev.key)) ev.preventDefault();
	};
	const handleKeyUp = (ev) => {
		ev.stopPropagation();
		let value = ev.target.value;
		let valueCleaned = "";
		for (let key of value) if (regexKeys.has(key)) valueCleaned += key;
		if (value !== valueCleaned) ev.target.value = valueCleaned;
		set(inputValue, valueCleaned, true);
		throttle(() => {
			setAutocompletedValue(valueCleaned);
		}, 150);
	};
	const handleFocus = (ev) => {
		ev.stopPropagation();
		set(showCalendar, true);
	};
	const handleBlur = (ev) => {
		ev.stopPropagation();
		if (avoidCloseOnBlur) {
			avoidCloseOnBlur = false;
			return;
		}
		set(showCalendar, false);
		if (get(fechaFocus) !== 0) {
			const value = (ev.target.value || "").trim();
			const acv = parseValue(value);
			if (value.length === 10 && acv.isCompleted) {
				changeFechaSelected(acv.fechaUnixAutocomplated);
				if ($$props.onChange) $$props.onChange();
			} else {
				ev.target.value = "";
				if ($$props.save && saveOn()) delete saveOn()[$$props.save];
			}
			set(fechaFocus, 0);
		}
	};
	user_effect(() => {
		if (saveOn() && $$props.save) {
			const fechaUnix = saveOn()[$$props.save];
			if (fechaUnix) {
				const value = makeFechaFormat(fechaUnix);
				setAutocompletedValue(value);
				set(inputValue, value, true);
				if (get(inputElement)) get(inputElement).value = value;
			} else {
				set(inputValue, "");
				if (get(inputElement)) get(inputElement).value = "";
				set(monthSelected, month_);
			}
			set(fechaSelected, fechaUnix || 0, true);
		}
	});
	let cN = user_derived(() => `${components_module_default.input} relative date-input-container` + (css() ? " " + css() : ""));
	var fragment = comment();
	var node = first_child(fragment);
	var consequent_2 = ($$anchor$1) => {
		var div = root_1();
		var div_1 = child(div);
		var div_2 = sibling(div_1, 2);
		var text = child(div_2, true);
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		var div_4 = sibling(div_3, 2);
		var div_5 = sibling(div_4, 2);
		var div_6 = child(div_5);
		var input = sibling(div_6, 2);
		remove_input_defaults(input);
		input.__keydown = handleKeyDown;
		input.__keyup = handleKeyUp;
		bind_this(input, ($$value) => set(inputElement, $$value), () => get(inputElement));
		reset(div_5);
		var node_1 = sibling(div_5, 2);
		var consequent_1 = ($$anchor$2) => {
			var div_7 = root_2();
			var div_8 = child(div_7);
			var button = child(div_8);
			button.__mousedown = (ev) => {
				ev.stopPropagation();
				avoidCloseOnBlur = true;
			};
			button.__click = (ev) => {
				ev.stopPropagation();
				changeMonth(-12);
			};
			var button_1 = sibling(button, 2);
			button_1.__mousedown = (ev) => {
				ev.stopPropagation();
				avoidCloseOnBlur = true;
			};
			button_1.__click = (ev) => {
				ev.stopPropagation();
				changeMonth(-1);
			};
			var div_9 = sibling(button_1, 2);
			var div_10 = child(div_9);
			var text_1 = child(div_10, true);
			reset(div_10);
			var div_11 = sibling(div_10, 2);
			var text_2 = child(div_11, true);
			reset(div_11);
			reset(div_9);
			var button_2 = sibling(div_9, 2);
			button_2.__mousedown = (ev) => {
				ev.stopPropagation();
				avoidCloseOnBlur = true;
			};
			button_2.__click = (ev) => {
				ev.stopPropagation();
				changeMonth(1);
			};
			var button_3 = sibling(button_2, 2);
			button_3.__mousedown = (ev) => {
				ev.stopPropagation();
				avoidCloseOnBlur = true;
			};
			button_3.__click = (ev) => {
				ev.stopPropagation();
				changeMonth(12);
			};
			reset(div_8);
			var div_12 = sibling(div_8, 2);
			each(sibling(child(div_12), 2), 17, () => weekDaysNames, index, ($$anchor$3, dayName) => {
				var div_13 = root_3();
				var text_3 = child(div_13, true);
				reset(div_13);
				template_effect(() => set_text(text_3, get(dayName).name));
				append($$anchor$3, div_13);
			});
			reset(div_12);
			each(sibling(div_12, 2), 17, () => get(semanasDias), index, ($$anchor$3, week) => {
				var div_14 = root_4();
				var div_15 = child(div_14);
				var text_4 = child(div_15, true);
				reset(div_15);
				each(sibling(div_15, 2), 17, () => get(week).weekDays, index, ($$anchor$4, day) => {
					const isOutMonth = user_derived(() => get(day).month !== get(monthSelected));
					const isSelected = user_derived(() => get(day).fechaUnix === get(fechaSelected));
					const isFocused = user_derived(() => get(day).fechaUnix === get(fechaFocus));
					const isToday = user_derived(() => fechaTodayUnix === get(day).fechaUnix);
					var button_4 = root_5();
					button_4.__click = (ev) => {
						ev.stopPropagation();
						changeFechaSelected(get(day).fechaUnix);
						set(showCalendar, false);
						set(fechaFocus, 0);
						avoidCloseOnBlur = false;
						if ($$props.onChange) $$props.onChange();
					};
					button_4.__mousedown = (ev) => {
						avoidCloseOnBlur = true;
						ev.stopPropagation();
					};
					var text_5 = child(button_4);
					var node_5 = sibling(text_5);
					var consequent = ($$anchor$5) => {
						append($$anchor$5, root_6());
					};
					if_block(node_5, ($$render) => {
						if (get(isToday)) $$render(consequent);
					});
					reset(button_4);
					template_effect(() => {
						set_class(button_4, 1, `relative aJs text-center flex items-center justify-center p-0 bg-transparent border-0 ${get(isOutMonth) ? "aJt" : ""} ${get(isSelected) ? "aJu" : ""} ${get(isFocused) ? "aJv" : ""}`, "aMy");
						set_text(text_5, `${get(day).day ?? ""} `);
					});
					append($$anchor$4, button_4);
				});
				reset(div_14);
				template_effect(() => set_text(text_4, get(week).week));
				append($$anchor$3, div_14);
			});
			reset(div_7);
			template_effect(() => {
				set_text(text_1, get(monthName).name);
				set_text(text_2, get(monthName).year);
			});
			event("mouseleave", div_7, (ev) => {
				ev.stopPropagation();
				if (get(inputElement) !== document.activeElement) {
					avoidCloseOnBlur = false;
					set(showCalendar, false);
					set(fechaFocus, 0);
				}
			});
			append($$anchor$2, div_7);
		};
		if_block(node_1, ($$render) => {
			if (get(showCalendar)) $$render(consequent_1);
		});
		reset(div);
		template_effect(() => {
			set_class(div, 1, clsx(get(cN)), "aMy");
			set_class(div_1, 1, clsx(components_module_default.input_lab_cell_left), "aMy");
			set_class(div_2, 1, clsx(components_module_default.input_lab), "aMy");
			set_text(text, label());
			set_class(div_3, 1, clsx(components_module_default.input_lab_cell_right), "aMy");
			set_class(div_4, 1, clsx(components_module_default.input_shadow_layer), "aMy");
			set_class(div_5, 1, `${components_module_default.input_div} flex w-full`, "aMy");
			set_class(div_6, 1, clsx(components_module_default.input_div_1), "aMy");
			set_class(input, 1, `w-full ${components_module_default.input_inp ?? ""} ff-mono ${(inputCss() || "") ?? ""}`, "aMy");
			set_value(input, get(inputValue));
			set_attribute(input, "placeholder", placeholder());
			input.disabled = disabled();
		});
		event("focus", input, handleFocus);
		event("blur", input, handleBlur);
		append($$anchor$1, div);
	};
	var alternate = ($$anchor$1) => {
		var div_17 = root_7();
		var div_18 = child(div_17);
		var div_19 = sibling(div_18, 2);
		var div_20 = child(div_19);
		var input_1 = sibling(div_20, 2);
		remove_input_defaults(input_1);
		input_1.__keydown = handleKeyDown;
		input_1.__keyup = handleKeyUp;
		bind_this(input_1, ($$value) => set(inputElement, $$value), () => get(inputElement));
		reset(div_19);
		var node_6 = sibling(div_19, 2);
		var consequent_4 = ($$anchor$2) => {
			var div_21 = root_8();
			var div_22 = child(div_21);
			var button_5 = child(div_22);
			button_5.__mousedown = (ev) => {
				ev.stopPropagation();
				avoidCloseOnBlur = true;
			};
			button_5.__click = (ev) => {
				ev.stopPropagation();
				changeMonth(-12);
			};
			var button_6 = sibling(button_5, 2);
			button_6.__mousedown = (ev) => {
				ev.stopPropagation();
				avoidCloseOnBlur = true;
			};
			button_6.__click = (ev) => {
				ev.stopPropagation();
				changeMonth(-1);
			};
			var div_23 = sibling(button_6, 2);
			var div_24 = child(div_23);
			var text_6 = child(div_24, true);
			reset(div_24);
			var div_25 = sibling(div_24, 2);
			var text_7 = child(div_25, true);
			reset(div_25);
			reset(div_23);
			var button_7 = sibling(div_23, 2);
			button_7.__mousedown = (ev) => {
				ev.stopPropagation();
				avoidCloseOnBlur = true;
			};
			button_7.__click = (ev) => {
				ev.stopPropagation();
				changeMonth(1);
			};
			var button_8 = sibling(button_7, 2);
			button_8.__mousedown = (ev) => {
				ev.stopPropagation();
				avoidCloseOnBlur = true;
			};
			button_8.__click = (ev) => {
				ev.stopPropagation();
				changeMonth(12);
			};
			reset(div_22);
			var div_26 = sibling(div_22, 2);
			each(sibling(child(div_26), 2), 17, () => weekDaysNames, index, ($$anchor$3, dayName) => {
				var div_27 = root_9();
				var text_8 = child(div_27, true);
				reset(div_27);
				template_effect(() => set_text(text_8, get(dayName).name));
				append($$anchor$3, div_27);
			});
			reset(div_26);
			each(sibling(div_26, 2), 17, () => get(semanasDias), index, ($$anchor$3, week) => {
				var div_28 = root_10();
				var div_29 = child(div_28);
				var text_9 = child(div_29, true);
				reset(div_29);
				each(sibling(div_29, 2), 17, () => get(week).weekDays, index, ($$anchor$4, day) => {
					const isOutMonth = user_derived(() => get(day).month !== get(monthSelected));
					const isSelected = user_derived(() => get(day).fechaUnix === get(fechaSelected));
					const isFocused = user_derived(() => get(day).fechaUnix === get(fechaFocus));
					const isToday = user_derived(() => fechaTodayUnix === get(day).fechaUnix);
					var button_9 = root_11();
					button_9.__click = (ev) => {
						ev.stopPropagation();
						changeFechaSelected(get(day).fechaUnix);
						set(showCalendar, false);
						set(fechaFocus, 0);
						avoidCloseOnBlur = false;
						if ($$props.onChange) $$props.onChange();
					};
					button_9.__mousedown = (ev) => {
						avoidCloseOnBlur = true;
						ev.stopPropagation();
					};
					var text_10 = child(button_9);
					var node_10 = sibling(text_10);
					var consequent_3 = ($$anchor$5) => {
						append($$anchor$5, root_12());
					};
					if_block(node_10, ($$render) => {
						if (get(isToday)) $$render(consequent_3);
					});
					reset(button_9);
					template_effect(() => {
						set_class(button_9, 1, `relative aJs text-center flex items-center justify-center p-0 bg-transparent border-0 ${get(isOutMonth) ? "aJt" : ""} ${get(isSelected) ? "aJu" : ""} ${get(isFocused) ? "aJv" : ""}`, "aMy");
						set_text(text_10, `${get(day).day ?? ""} `);
					});
					append($$anchor$4, button_9);
				});
				reset(div_28);
				template_effect(() => set_text(text_9, get(week).week));
				append($$anchor$3, div_28);
			});
			reset(div_21);
			template_effect(() => {
				set_text(text_6, get(monthName).name);
				set_text(text_7, get(monthName).year);
			});
			event("mouseleave", div_21, (ev) => {
				ev.stopPropagation();
				if (get(inputElement) !== document.activeElement) {
					avoidCloseOnBlur = false;
					set(showCalendar, false);
					set(fechaFocus, 0);
				}
			});
			append($$anchor$2, div_21);
		};
		if_block(node_6, ($$render) => {
			if (get(showCalendar)) $$render(consequent_4);
		});
		reset(div_17);
		template_effect(() => {
			set_class(div_17, 1, `${components_module_default.input} no-label relative date-input-container` + (css() ? " " + css() : ""), "aMy");
			set_class(div_18, 1, clsx(components_module_default.input_shadow_layer), "aMy");
			set_class(div_19, 1, `${components_module_default.input_div} flex w-full`, "aMy");
			set_class(div_20, 1, clsx(components_module_default.input_div_1), "aMy");
			set_class(input_1, 1, `w-full ${components_module_default.input_inp ?? ""} ff-mono ${(inputCss() || "") ?? ""}`, "aMy");
			set_value(input_1, get(inputValue));
			set_attribute(input_1, "placeholder", placeholder());
			input_1.disabled = disabled();
		});
		event("focus", input_1, handleFocus);
		event("blur", input_1, handleBlur);
		append($$anchor$1, div_17);
	};
	if_block(node, ($$render) => {
		if (label()) $$render(consequent_2);
		else $$render(alternate, false);
	});
	append($$anchor, fragment);
	pop();
}
delegate([
	"keydown",
	"keyup",
	"mousedown",
	"click"
]);
export { DateInput as t };
