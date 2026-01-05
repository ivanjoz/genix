import { $ as proxy, A as if_block, B as tick, C as clsx, Ct as __reExport, D as html, E as snippet, G as user_effect, L as delegate, M as append, N as comment, O as each, P as from_html, Q as sibling, R as event, S as set_class, T as component, Tt as __toESM, V as untrack, W as template_effect, X as child, Y as $window, Z as first_child, _ as remove_input_defaults, _t as reset, a as prop, at as state, d as bind_window_size, f as bind_this, gt as next, j as set_text, k as index, lt as pop, mt as snapshot, p as bind_checked, r as onMount, rt as set, st as user_derived, ut as push, v as set_attribute, vt as noop, x as set_style, xt as __commonJSMin, y as set_value, z as get } from "../chunks/CTB4HzdN.js";
import "../chunks/D25ese0K.js";
import "../chunks/2yAUi6s0.js";
import { t as browser } from "../chunks/BemFyt0E.js";
import { b as formatN, c as openModal, h as POST, i as closeModal, r as closeAllModals, s as imagesToUpload, t as Core } from "../chunks/BwrZ3UQO.js";
import { a as throttle } from "../chunks/DwKmqXYU.js";
import { c as parseSVG, n as Loading, r as Notify, t as ConfirmWarn } from "../chunks/DOFkf9MZ.js";
import { t as components_module_default } from "../chunks/cyfMRGwx.js";
import { t as Input$1 } from "../chunks/N4ddDIuf.js";
import { t as OptionsStrip } from "../chunks/CGFlWCTf.js";
import { t as Page } from "../chunks/GeoukCMt.js";
import { t as VTable } from "../chunks/BCJF-hdW.js";
import "../chunks/DnSz4jvr.js";
import { t as Modal } from "../chunks/iQLQpuQM.js";
import { t as Layer } from "../chunks/C1AypaHL.js";
import { t as SearchSelect } from "../chunks/BOmyLNB5.js";
import { t as SearchCard } from "../chunks/BHNEJas0.js";
import { t as CheckboxOptions } from "../chunks/Bmy_fcV9.js";
import { t as ImageUploader } from "../chunks/5nFXSVM4.js";
import "../chunks/o-eLsTb7.js";
import { a as productoAtributos, i as postProducto, n as ProductosService, r as postListaRegistros, t as ListasCompartidasService } from "../chunks/DAmFAq63.js";
const editorTextColors = [
	"#000000",
	"#333333",
	"#666666",
	"#8B0000",
	"#CC0000",
	"#DC143C",
	"#800080",
	"#4B0082",
	"#A020F0",
	"#696969",
	"#483D8B",
	"#B8860B",
	"#000080",
	"#0047AB",
	"#007BFF",
	"#008080",
	"#228B22",
	"#556B2F",
	"#CC5500",
	"#8B4513",
	"#996600",
	"#FF1493",
	"#00CED1",
	"#FF4500"
];
const editorBackgroundColors = [
	"#000000",
	"#333333",
	"#666666",
	"#8B0000",
	"#CC0000",
	"#DC143C",
	"#800080",
	"#4B0082",
	"#A020F0",
	"#696969",
	"#483D8B",
	"#B8860B",
	"#000080",
	"#0047AB",
	"#007BFF",
	"#008080",
	"#228B22",
	"#556B2F",
	"#CC5500",
	"#8B4513",
	"#996600",
	"#FF1493",
	"#00CED1",
	"#FF4500"
];
const editorTextSizes = [
	{
		id: 12,
		name: "12"
	},
	{
		id: 14,
		name: "14"
	},
	{
		id: 16,
		name: "16"
	},
	{
		id: 18,
		name: "18"
	},
	{
		id: 20,
		name: "20"
	}
];
const TextSizeIcon = `<svg width="800px" height="800px" viewBox="0 0 512 512" xmlns="http://www.w3.org/2000/svg">
  <polygon fill="var(--ci-primary-color, #000000)" points="176 184 208 184 208 136 304 136 304 368 264 368 264 400 408 400 408 368 368 368 368 136 464 136 464 184 496 184 496 104 176 104 176 184" class="ci-primary"/>
  <polygon fill="var(--ci-primary-color, #000000)" points="16 280 48 280 48 248 104 248 104 400 72 400 72 432 184 432 184 400 152 400 152 248 216 248 216 280 248 280 248 216 16 216 16 280" class="ci-primary"/>
</svg>`;
const TextColorIcon = `<svg fill="#000000" width="800px" height="800px" viewBox="0 0 14 14" role="img" focusable="false" aria-hidden="true" xmlns="http://www.w3.org/2000/svg"><path d="M 7.5291661,11.795909 C 7.4168129,11.419456 7.3406864,10.225625 7.3406864,9.29222 c 0,-0.11438 -0.029767,-0.221667 -0.081573,-0.314893 0.051933,-0.115773 0.08132,-0.24358 0.08132,-0.378226 l 0,-1.709364 c 0,-0.511733 -0.416226,-0.927959 -0.9279585,-0.927959 l -0.8772919,0 C 5.527203,5.856265 5.52163,5.751005 5.518336,5.648406 5.514666,5.556066 5.513396,5.470313 5.513016,5.385826 5.511876,5.296776 5.5132694,5.224073 5.517196,5.160866 5.524666,5.024193 5.541009,4.891827 5.565076,4.773647 5.591043,4.646981 5.619669,4.564774 5.630689,4.535134 c 0.0019,-0.0052 0.0038,-0.01013 0.00557,-0.01533 0.00709,-0.02039 0.0133,-0.03559 0.017227,-0.04446 C 6.0127121,3.789698 5.750766,2.938499 5.0665137,2.5737 4.8642273,2.466034 4.6367344,2.409034 4.4084814,2.408147 4.1801018,2.409034 3.9526089,2.466037 3.7504492,2.5737 3.066197,2.938499 2.8042508,3.789698 3.1634768,4.475344 c 0.00393,0.0087 0.01026,0.02394 0.017227,0.04446 0.00177,0.0052 0.00367,0.01013 0.00557,0.01533 0.01102,0.02951 0.039647,0.111847 0.065613,0.238513 0.024067,0.11818 0.040533,0.250546 0.04788,0.387219 0.00393,0.06321 0.00532,0.135914 0.00418,0.22496 -5.066e-4,0.08449 -0.00165,0.17024 -0.00532,0.26258 -0.00329,0.102599 -0.00887,0.207859 -0.016847,0.313372 l -0.8772919,0 c -0.5117324,0 -0.9279584,0.416226 -0.9279584,0.927959 l 0,1.709364 c 0,0.134646 0.029387,0.262453 0.08132,0.378226 -0.051807,0.09323 -0.081573,0.200513 -0.081573,0.314893 0,0.933278 -0.076126,2.127236 -0.1884796,2.503689 C 1.0571435,11.985782 1.0131902,12.254315 1.0562568,12.453434 1.1748167,13 1.7477291,13 1.9359554,13 c 0.437506,0 1.226258,-0.07676 1.2595712,-0.08005 0.05092,-0.0051 0.1001932,-0.01596 0.1468065,-0.03179 0.049907,0.01241 0.1018398,0.01913 0.1546597,0.01925 l 0.9114918,0.0044 0.9114918,-0.0044 c 0.05282,-1.27e-4 0.1047532,-0.007 0.1546598,-0.01925 0.046613,0.01583 0.095886,0.02673 0.1468064,0.03179 C 5.6547556,12.92315 6.4436346,13 6.8810138,13 c 0.1882264,0 0.7612654,0 0.8796986,-0.546566 0.043067,-0.199119 -7.6e-4,-0.467652 -0.2315463,-0.657525 z m -1.833117,0.502486 -0.3480794,-1.518478 -0.1741664,1.503658 -1.6846638,-7.6e-4 -0.3680927,-0.885399 0,0.900979 c 0,0 -1.7672504,0.173279 -1.3861111,0 0.3811394,-0.173154 0.3811394,-2.980082 0.3811394,-2.980082 l 2.2924095,0 2.2924095,0 c 0,0 0,2.806928 0.3811394,2.980082 0.381266,0.173279 -1.3859844,0 -1.3859844,0 z M 10.219055,1 7.3387864,1 5.8932688,5.377719 l 0.9449318,0 c 0.3536527,0 0.6674055,0.17138 0.8650052,0.434593 l 0.04864,-0.18392 0.9107318,-2.702555 0.2962729,-0.0016 0.9543051,2.889769 -2.2085564,0 C 7.839499,5.994632 7.9204389,6.217692 7.9204389,6.459878 l 0,1.257038 2.3962751,0 0.423193,1.60917 2.218563,0 L 10.219055,1 Z"/></svg>`;
const TextBackgroudColor = `<svg width="800px" height="800px" viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg">
  <title>text-color-solid</title>
  <g id="Layer_2" data-name="Layer 2">
    <g id="invisible_box" data-name="invisible box">
      <rect width="48" height="48" fill="none"/>
    </g>
    <g id="Q3_icons" data-name="Q3 icons">
      <path d="M43,3H5A2,2,0,0,0,3,5V43a2,2,0,0,0,2,2H43a2,2,0,0,0,2-2V5A2,2,0,0,0,43,3ZM33.4,37l-2.6-6H17.2l-2.6,6H10.2L21.5,11h5L37.8,37ZM18.9,27H29.1L24,14.9Z"/>
    </g>
  </g>
</svg>`;
var extendStatics = function(d$1, b$1) {
	extendStatics = Object.setPrototypeOf || { __proto__: [] } instanceof Array && function(d$2, b$2) {
		d$2.__proto__ = b$2;
	} || function(d$2, b$2) {
		for (var p$1 in b$2) if (Object.prototype.hasOwnProperty.call(b$2, p$1)) d$2[p$1] = b$2[p$1];
	};
	return extendStatics(d$1, b$1);
};
function __extends(d$1, b$1) {
	if (typeof b$1 !== "function" && b$1 !== null) throw new TypeError("Class extends value " + String(b$1) + " is not a constructor or null");
	extendStatics(d$1, b$1);
	function __() {
		this.constructor = d$1;
	}
	d$1.prototype = b$1 === null ? Object.create(b$1) : (__.prototype = b$1.prototype, new __());
}
var __assign = function() {
	__assign = Object.assign || function __assign$1(t$1) {
		for (var s$1, i$1 = 1, n$1 = arguments.length; i$1 < n$1; i$1++) {
			s$1 = arguments[i$1];
			for (var p$1 in s$1) if (Object.prototype.hasOwnProperty.call(s$1, p$1)) t$1[p$1] = s$1[p$1];
		}
		return t$1;
	};
	return __assign.apply(this, arguments);
};
function __generator(thisArg, body) {
	var _ = {
		label: 0,
		sent: function() {
			if (t$1[0] & 1) throw t$1[1];
			return t$1[1];
		},
		trys: [],
		ops: []
	}, f$1, y$1, t$1, g$1 = Object.create((typeof Iterator === "function" ? Iterator : Object).prototype);
	return g$1.next = verb(0), g$1["throw"] = verb(1), g$1["return"] = verb(2), typeof Symbol === "function" && (g$1[Symbol.iterator] = function() {
		return this;
	}), g$1;
	function verb(n$1) {
		return function(v$1) {
			return step([n$1, v$1]);
		};
	}
	function step(op) {
		if (f$1) throw new TypeError("Generator is already executing.");
		while (g$1 && (g$1 = 0, op[0] && (_ = 0)), _) try {
			if (f$1 = 1, y$1 && (t$1 = op[0] & 2 ? y$1["return"] : op[0] ? y$1["throw"] || ((t$1 = y$1["return"]) && t$1.call(y$1), 0) : y$1.next) && !(t$1 = t$1.call(y$1, op[1])).done) return t$1;
			if (y$1 = 0, t$1) op = [op[0] & 2, t$1.value];
			switch (op[0]) {
				case 0:
				case 1:
					t$1 = op;
					break;
				case 4:
					_.label++;
					return {
						value: op[1],
						done: false
					};
				case 5:
					_.label++;
					y$1 = op[1];
					op = [0];
					continue;
				case 7:
					op = _.ops.pop();
					_.trys.pop();
					continue;
				default:
					if (!(t$1 = _.trys, t$1 = t$1.length > 0 && t$1[t$1.length - 1]) && (op[0] === 6 || op[0] === 2)) {
						_ = 0;
						continue;
					}
					if (op[0] === 3 && (!t$1 || op[1] > t$1[0] && op[1] < t$1[3])) {
						_.label = op[1];
						break;
					}
					if (op[0] === 6 && _.label < t$1[1]) {
						_.label = t$1[1];
						t$1 = op;
						break;
					}
					if (t$1 && _.label < t$1[2]) {
						_.label = t$1[2];
						_.ops.push(op);
						break;
					}
					if (t$1[2]) _.ops.pop();
					_.trys.pop();
					continue;
			}
			op = body.call(thisArg, _);
		} catch (e$1) {
			op = [6, e$1];
			y$1 = 0;
		} finally {
			f$1 = t$1 = 0;
		}
		if (op[0] & 5) throw op[1];
		return {
			value: op[0] ? op[1] : void 0,
			done: true
		};
	}
}
function __values(o$1) {
	var s$1 = typeof Symbol === "function" && Symbol.iterator, m$1 = s$1 && o$1[s$1], i$1 = 0;
	if (m$1) return m$1.call(o$1);
	if (o$1 && typeof o$1.length === "number") return { next: function() {
		if (o$1 && i$1 >= o$1.length) o$1 = void 0;
		return {
			value: o$1 && o$1[i$1++],
			done: !o$1
		};
	} };
	throw new TypeError(s$1 ? "Object is not iterable." : "Symbol.iterator is not defined.");
}
function __read(o$1, n$1) {
	var m$1 = typeof Symbol === "function" && o$1[Symbol.iterator];
	if (!m$1) return o$1;
	var i$1 = m$1.call(o$1), r$1, ar = [], e$1;
	try {
		while ((n$1 === void 0 || n$1-- > 0) && !(r$1 = i$1.next()).done) ar.push(r$1.value);
	} catch (error) {
		e$1 = { error };
	} finally {
		try {
			if (r$1 && !r$1.done && (m$1 = i$1["return"])) m$1.call(i$1);
		} finally {
			if (e$1) throw e$1.error;
		}
	}
	return ar;
}
function __spreadArray(to, from, pack) {
	if (pack || arguments.length === 2) {
		for (var i$1 = 0, l$1 = from.length, ar; i$1 < l$1; i$1++) if (ar || !(i$1 in from)) {
			if (!ar) ar = Array.prototype.slice.call(from, 0, i$1);
			ar[i$1] = from[i$1];
		}
	}
	return to.concat(ar || Array.prototype.slice.call(from));
}
function createContentModelDocument(defaultFormat) {
	var result = {
		blockGroupType: "Document",
		blocks: []
	};
	if (defaultFormat) result.format = defaultFormat;
	return result;
}
function isBlockEmpty(block) {
	switch (block.blockType) {
		case "Paragraph": return block.segments.length == 0;
		case "Table": return block.rows.every(function(row) {
			return row.cells.length == 0;
		});
		case "BlockGroup": return isBlockGroupEmpty(block);
		case "Entity": return false;
		default: return false;
	}
}
function isBlockGroupEmpty(group) {
	switch (group.blockGroupType) {
		case "FormatContainer": return group.tagName == "div" ? false : group.blocks.every(isBlockEmpty);
		case "ListItem": return group.blocks.every(isBlockEmpty);
		case "Document":
		case "General":
		case "TableCell": return false;
		default: return true;
	}
}
function isSegmentEmpty(segment, treatAnchorAsNotEmpty) {
	var _a$5;
	switch (segment.segmentType) {
		case "Text": return !segment.text && (!treatAnchorAsNotEmpty || !((_a$5 = segment.link) === null || _a$5 === void 0 ? void 0 : _a$5.format.name));
		default: return false;
	}
}
function isEmpty(model) {
	if (isBlockGroup(model)) return isBlockGroupEmpty(model);
	else if (isBlock(model)) return isBlockEmpty(model);
	else if (isSegment(model)) return isSegmentEmpty(model);
	return false;
}
function isSegment(model) {
	return typeof model.segmentType === "string";
}
function isBlock(model) {
	return typeof model.blockType === "string";
}
function isBlockGroup(model) {
	return typeof model.blockGroupType === "string";
}
var ListFormatsToMove = [
	"marginRight",
	"marginLeft",
	"paddingRight",
	"paddingLeft"
];
var ListFormatsToKeep = [
	"direction",
	"textAlign",
	"htmlAlign"
];
var ListFormats = ListFormatsToMove.concat(ListFormatsToKeep);
var ParagraphFormats = [
	"backgroundColor",
	"direction",
	"textAlign",
	"htmlAlign",
	"lineHeight",
	"textIndent",
	"marginTop",
	"marginRight",
	"marginBottom",
	"marginLeft",
	"paddingTop",
	"paddingRight",
	"paddingBottom",
	"paddingLeft"
];
function copyFormat(targetFormat, sourceFormat, formatKeys, deleteOriginalFormat) {
	var e_1, _a$5, _b$1;
	try {
		for (var formatKeys_1 = __values(formatKeys), formatKeys_1_1 = formatKeys_1.next(); !formatKeys_1_1.done; formatKeys_1_1 = formatKeys_1.next()) {
			var key = formatKeys_1_1.value;
			if (sourceFormat[key] !== void 0) {
				Object.assign(targetFormat, (_b$1 = {}, _b$1[key] = sourceFormat[key], _b$1));
				if (deleteOriginalFormat) delete sourceFormat[key];
			}
		}
	} catch (e_1_1) {
		e_1 = { error: e_1_1 };
	} finally {
		try {
			if (formatKeys_1_1 && !formatKeys_1_1.done && (_a$5 = formatKeys_1.return)) _a$5.call(formatKeys_1);
		} finally {
			if (e_1) throw e_1.error;
		}
	}
}
function mutateBlock(block) {
	if (block.cachedElement) delete block.cachedElement;
	if (isTable(block)) block.rows.forEach(function(row) {
		delete row.cachedElement;
	});
	else if (isListItem(block)) block.levels.forEach(function(level) {
		return delete level.cachedElement;
	});
	return block;
}
function mutateSegments(paragraph, segments) {
	var mutablePara = mutateBlock(paragraph);
	var result = [
		mutablePara,
		[],
		[]
	];
	if (segments) segments.forEach(function(segment) {
		var index$1 = paragraph.segments.indexOf(segment);
		if (index$1 >= 0) {
			result[1].push(mutablePara.segments[index$1]);
			result[2].push(index$1);
		}
	});
	return result;
}
function mutateSegment(paragraph, segment, callback) {
	var _a$5;
	var _b$1 = __read(mutateSegments(paragraph, [segment]), 3), mutablePara = _b$1[0], mutableSegments = _b$1[1], indexes = _b$1[2];
	var mutableSegment = mutableSegments[0] == segment ? mutableSegments[0] : null;
	if (callback && mutableSegment) callback(mutableSegments[0], mutablePara, indexes[0]);
	return [
		mutablePara,
		mutableSegment,
		(_a$5 = indexes[0]) !== null && _a$5 !== void 0 ? _a$5 : -1
	];
}
function isTable(obj) {
	return obj.blockType == "Table";
}
function isListItem(obj) {
	return obj.blockGroupType == "ListItem";
}
function getObjectKeys(obj) {
	return Object.keys(obj);
}
function areSameFormats(f1, f2) {
	if (f1 == f2) return true;
	else {
		var keys1 = getObjectKeys(f1);
		var keys2 = getObjectKeys(f2);
		return keys1.length == keys2.length && keys1.every(function(key) {
			return f1[key] == f2[key];
		});
	}
}
function createBr(format) {
	return {
		segmentType: "Br",
		format: __assign({}, format)
	};
}
var WHITESPACE_PRE_VALUES = [
	"pre",
	"pre-wrap",
	"break-spaces"
];
function isWhiteSpacePreserved(whiteSpace) {
	return !!whiteSpace && WHITESPACE_PRE_VALUES.indexOf(whiteSpace) >= 0;
}
var SPACE_TEXT_REGEX = /^[\r\n\t ]*$/;
function hasSpacesOnly(txt) {
	return SPACE_TEXT_REGEX.test(txt);
}
var SPACE = " ";
var NONE_BREAK_SPACE = "\xA0";
var LEADING_SPACE_REGEX = /^\u0020+/;
var TRAILING_SPACE_REGEX = /\u0020+$/;
function normalizeAllSegments(paragraph) {
	var context = resetNormalizeSegmentContext();
	paragraph.segments.forEach(function(segment) {
		normalizeSegment(paragraph, segment, context);
	});
	normalizeTextSegments(paragraph, context.textSegments, context.lastInlineSegment);
	normalizeLastTextSegment(paragraph, context.lastTextSegment, context.lastInlineSegment);
}
function normalizeSingleSegment(paragraph, segment, ignoreTrailingSpaces) {
	if (ignoreTrailingSpaces === void 0) ignoreTrailingSpaces = false;
	var context = resetNormalizeSegmentContext();
	context.ignoreTrailingSpaces = ignoreTrailingSpaces;
	normalizeSegment(paragraph, segment, context);
}
function resetNormalizeSegmentContext(context) {
	return Object.assign(context !== null && context !== void 0 ? context : {}, {
		textSegments: [],
		ignoreLeadingSpaces: true,
		ignoreTrailingSpaces: true,
		lastInlineSegment: void 0,
		lastTextSegment: void 0
	});
}
function normalizeSegment(paragraph, segment, context) {
	switch (segment.segmentType) {
		case "Br":
			normalizeTextSegments(paragraph, context.textSegments, context.lastInlineSegment);
			normalizeLastTextSegment(paragraph, context.lastTextSegment, context.lastInlineSegment);
			resetNormalizeSegmentContext(context);
			break;
		case "Entity":
		case "General":
		case "Image":
			context.lastInlineSegment = segment;
			context.ignoreLeadingSpaces = false;
			break;
		case "Text":
			context.textSegments.push(segment);
			context.lastInlineSegment = segment;
			context.lastTextSegment = segment;
			var first = segment.text.substring(0, 1);
			var last = segment.text.substr(-1);
			if (!hasSpacesOnly(segment.text)) {
				if (first == SPACE) mutateSegment(paragraph, segment, function(textSegment) {
					textSegment.text = textSegment.text.replace(LEADING_SPACE_REGEX, context.ignoreLeadingSpaces ? "" : NONE_BREAK_SPACE);
				});
				if (last == SPACE) mutateSegment(paragraph, segment, function(textSegment) {
					textSegment.text = textSegment.text.replace(TRAILING_SPACE_REGEX, context.ignoreTrailingSpaces ? SPACE : NONE_BREAK_SPACE);
				});
			}
			context.ignoreLeadingSpaces = last == SPACE;
			break;
	}
}
function normalizeTextSegments(paragraph, segments, lastInlineSegment) {
	segments.forEach(function(segment) {
		if (segment != lastInlineSegment) {
			var text_1 = segment.text;
			if (text_1.substr(-1) == NONE_BREAK_SPACE && text_1.length > 1 && text_1.substr(-2, 1) != SPACE) mutateSegment(paragraph, segment, function(textSegment) {
				textSegment.text = text_1.substring(0, text_1.length - 1) + SPACE;
			});
		}
	});
}
function normalizeLastTextSegment(paragraph, segment, lastInlineSegment) {
	if (segment && segment == lastInlineSegment && (segment === null || segment === void 0 ? void 0 : segment.text.substr(-1)) == SPACE) mutateSegment(paragraph, segment, function(textSegment) {
		textSegment.text = textSegment.text.replace(TRAILING_SPACE_REGEX, "");
	});
}
function normalizeParagraph(paragraph) {
	var segments = paragraph.segments;
	if (!paragraph.isImplicit && segments.length > 0) {
		var last = segments[segments.length - 1];
		var secondLast = segments[segments.length - 2];
		if (last.segmentType == "SelectionMarker" && (!secondLast || secondLast.segmentType == "Br")) mutateBlock(paragraph).segments.push(createBr(last.format));
		else if (segments.length > 1 && segments[segments.length - 1].segmentType == "Br") {
			var noMarkerSegments = segments.filter(function(x$1) {
				return x$1.segmentType != "SelectionMarker";
			});
			if (noMarkerSegments.length > 1 && noMarkerSegments[noMarkerSegments.length - 2].segmentType != "Br") mutateBlock(paragraph).segments.pop();
		}
		normalizeParagraphStyle(paragraph);
	}
	if (!isWhiteSpacePreserved(paragraph.format.whiteSpace)) normalizeAllSegments(paragraph);
	removeEmptyLinks(paragraph);
	removeEmptySegments(paragraph);
	moveUpSegmentFormat(paragraph);
}
function normalizeParagraphStyle(paragraph) {
	if (paragraph.format.whiteSpace && paragraph.segments.every(function(seg) {
		return seg.segmentType == "Br" || seg.segmentType == "SelectionMarker";
	})) delete mutateBlock(paragraph).format.whiteSpace;
}
function removeEmptySegments(block) {
	for (var j$1 = block.segments.length - 1; j$1 >= 0; j$1--) if (isSegmentEmpty(block.segments[j$1], true)) mutateBlock(block).segments.splice(j$1, 1);
}
function removeEmptyLinks(paragraph) {
	var marker = paragraph.segments.find(function(x$1) {
		return x$1.segmentType == "SelectionMarker";
	});
	if (marker) {
		var markerIndex = paragraph.segments.indexOf(marker);
		var prev = paragraph.segments[markerIndex - 1];
		var next$1 = paragraph.segments[markerIndex + 1];
		if (prev && !prev.link && areSameFormats(prev.format, marker.format) && (!next$1 || !next$1.link && areSameFormats(next$1.format, marker.format)) && marker.link || !prev && marker.link && next$1 && !next$1.link && areSameFormats(next$1.format, marker.format)) mutateSegment(paragraph, marker, function(mutableMarker) {
			delete mutableMarker.link;
		});
	}
}
var formatsToMoveUp = [
	"fontFamily",
	"fontSize",
	"textColor"
];
function moveUpSegmentFormat(paragraph) {
	if (!paragraph.decorator) {
		var segments_1 = paragraph.segments.filter(function(x$1) {
			return x$1.segmentType != "SelectionMarker";
		});
		var target_1 = paragraph.segmentFormat || {};
		var changed_1 = false;
		formatsToMoveUp.forEach(function(key) {
			changed_1 = internalMoveUpSegmentFormat(segments_1, target_1, key) || changed_1;
		});
		if (changed_1) mutateBlock(paragraph).segmentFormat = target_1;
	}
}
function internalMoveUpSegmentFormat(segments, target, formatKey) {
	var _a$5;
	var firstFormat = (_a$5 = segments[0]) === null || _a$5 === void 0 ? void 0 : _a$5.format;
	if ((firstFormat === null || firstFormat === void 0 ? void 0 : firstFormat[formatKey]) && segments.every(function(segment) {
		return segment.format[formatKey] == firstFormat[formatKey];
	}) && target[formatKey] != firstFormat[formatKey]) {
		target[formatKey] = firstFormat[formatKey];
		return true;
	} else return false;
}
function setParagraphNotImplicit(block) {
	if (block.blockType == "Paragraph" && block.isImplicit) mutateBlock(block).isImplicit = false;
}
function unwrapBlock(parent, groupToUnwrap, formatsToKeep) {
	var _a$5;
	var _b$1, _c$1;
	var index$1 = (_b$1 = parent === null || parent === void 0 ? void 0 : parent.blocks.indexOf(groupToUnwrap)) !== null && _b$1 !== void 0 ? _b$1 : -1;
	if (index$1 >= 0) {
		groupToUnwrap.blocks.forEach(setParagraphNotImplicit);
		if (parent) (_c$1 = mutateBlock(parent)) === null || _c$1 === void 0 || (_a$5 = _c$1.blocks).splice.apply(_a$5, __spreadArray([index$1, 1], __read(groupToUnwrap.blocks.map(function(x$1) {
			var mutableBlock = mutateBlock(x$1);
			if (formatsToKeep) copyFormat(mutableBlock.format, groupToUnwrap.format, formatsToKeep);
			return mutableBlock;
		})), false));
	}
}
function normalizeContentModel(group) {
	for (var i$1 = group.blocks.length - 1; i$1 >= 0; i$1--) {
		var block = group.blocks[i$1];
		switch (block.blockType) {
			case "BlockGroup":
				if (block.blockGroupType == "ListItem" && block.levels.length == 0) {
					i$1 += block.blocks.length;
					unwrapBlock(group, block, ListFormats);
				} else normalizeContentModel(block);
				break;
			case "Paragraph":
				normalizeParagraph(block);
				break;
			case "Table":
				for (var r$1 = 0; r$1 < block.rows.length; r$1++) for (var c$1 = 0; c$1 < block.rows[r$1].cells.length; c$1++) if (block.rows[r$1].cells[c$1]) normalizeContentModel(block.rows[r$1].cells[c$1]);
				break;
		}
		if (isBlockEmpty(block)) mutateBlock(group).blocks.splice(i$1, 1);
	}
}
function domToContentModel(root$12, context) {
	var _a$5;
	var model = createContentModelDocument(context.defaultFormat);
	if (((_a$5 = context.selection) === null || _a$5 === void 0 ? void 0 : _a$5.type) == "range" && context.selection.isReverted) model.hasRevertedRangeSelection = true;
	if (context.domIndexer && context.allowCacheElement) model.persistCache = true;
	context.elementProcessors.child(model, root$12, context);
	normalizeContentModel(model);
	return model;
}
function isNodeOfType(node, expectedType) {
	return !!node && node.nodeType == Node[expectedType];
}
function toArray(collection) {
	return [].slice.call(collection);
}
function contentModelToDom(doc, root$12, model, context) {
	context.modelHandlers.blockGroupChildren(doc, root$12, model, context);
	var range = extractSelectionRange(doc, context);
	if (model.hasRevertedRangeSelection && (range === null || range === void 0 ? void 0 : range.type) == "range") range.isReverted = true;
	if (context.domIndexer && context.allowCacheElement) model.persistCache = true;
	root$12.normalize();
	return range;
}
function extractSelectionRange(doc, context) {
	var _a$5 = context.regularSelection, start = _a$5.start, end = _a$5.end, tableSelection = context.tableSelection, imageSelection = context.imageSelection;
	var startPosition;
	var endPosition;
	if (imageSelection) return imageSelection;
	else if ((startPosition = start && calcPosition(start)) && (endPosition = end && calcPosition(end))) {
		var range = doc.createRange();
		range.setStart(startPosition.container, startPosition.offset);
		range.setEnd(endPosition.container, endPosition.offset);
		return {
			type: "range",
			range,
			isReverted: false
		};
	} else if (tableSelection) return tableSelection;
	else return null;
}
function calcPosition(pos) {
	var _a$5, _b$1, _c$1, _d$1, _e$1;
	var result;
	if (pos.block) {
		if (!pos.segment) result = {
			container: pos.block,
			offset: 0
		};
		else if (isNodeOfType(pos.segment, "TEXT_NODE")) result = {
			container: pos.segment,
			offset: (_c$1 = (_a$5 = pos.offset) !== null && _a$5 !== void 0 ? _a$5 : (_b$1 = pos.segment.nodeValue) === null || _b$1 === void 0 ? void 0 : _b$1.length) !== null && _c$1 !== void 0 ? _c$1 : 0
		};
		else if (pos.segment.parentNode) result = {
			container: pos.segment.parentNode,
			offset: toArray(pos.segment.parentNode.childNodes).indexOf(pos.segment) + 1
		};
	}
	if (result && isNodeOfType(result.container, "DOCUMENT_FRAGMENT_NODE")) {
		var childNodes = result.container.childNodes;
		if (childNodes.length > result.offset) result = {
			container: childNodes[result.offset],
			offset: 0
		};
		else if (result.container.lastChild) {
			var container = result.container.lastChild;
			result = {
				container,
				offset: isNodeOfType(container, "TEXT_NODE") ? (_e$1 = (_d$1 = container.nodeValue) === null || _d$1 === void 0 ? void 0 : _d$1.length) !== null && _e$1 !== void 0 ? _e$1 : 0 : container.childNodes.length
			};
		} else result = void 0;
	}
	return result;
}
var TextForHR = "________________________________________";
var defaultCallbacks = {
	onDivider: function(divider) {
		return divider.tagName == "hr" ? TextForHR : "";
	},
	onEntityBlock: function() {
		return "";
	},
	onEntitySegment: function(entity) {
		var _a$5;
		return (_a$5 = entity.wrapper.textContent) !== null && _a$5 !== void 0 ? _a$5 : "";
	},
	onGeneralSegment: function(segment) {
		var _a$5;
		return (_a$5 = segment.element.textContent) !== null && _a$5 !== void 0 ? _a$5 : "";
	},
	onImage: function() {
		return " ";
	},
	onText: function(text) {
		return text.text;
	},
	onParagraph: function() {
		return true;
	},
	onTable: function() {
		return true;
	},
	onBlockGroup: function() {
		return true;
	}
};
function contentModelToText(model, separator, callbacks) {
	if (separator === void 0) separator = "\r\n";
	var textArray = [];
	contentModelToTextArray(model, textArray, Object.assign({}, defaultCallbacks, callbacks));
	return textArray.join(separator);
}
function contentModelToTextArray(group, textArray, callbacks) {
	if (callbacks.onBlockGroup(group)) group.blocks.forEach(function(block) {
		switch (block.blockType) {
			case "Paragraph":
				if (callbacks.onParagraph(block)) {
					var text_1 = "";
					block.segments.forEach(function(segment) {
						switch (segment.segmentType) {
							case "Br":
								textArray.push(text_1);
								text_1 = "";
								break;
							case "Entity":
								text_1 += callbacks.onEntitySegment(segment);
								break;
							case "General":
								text_1 += callbacks.onGeneralSegment(segment);
								break;
							case "Text":
								text_1 += callbacks.onText(segment);
								break;
							case "Image":
								text_1 += callbacks.onImage(segment);
								break;
						}
					});
					if (text_1) textArray.push(text_1);
				}
				break;
			case "Divider":
				textArray.push(callbacks.onDivider(block));
				break;
			case "Entity":
				textArray.push(callbacks.onEntityBlock(block));
				break;
			case "Table":
				if (callbacks.onTable(block)) block.rows.forEach(function(row) {
					return row.cells.forEach(function(cell) {
						contentModelToTextArray(cell, textArray, callbacks);
					});
				});
				break;
			case "BlockGroup":
				contentModelToTextArray(block, textArray, callbacks);
				break;
		}
	});
}
function addBlock(group, block) {
	group.blocks.push(block);
}
function createParagraph(isImplicit, blockFormat, segmentFormat, decorator) {
	var result = {
		blockType: "Paragraph",
		segments: [],
		format: __assign({}, blockFormat)
	};
	if (segmentFormat && Object.keys(segmentFormat).length > 0) result.segmentFormat = __assign({}, segmentFormat);
	if (isImplicit) result.isImplicit = true;
	if (decorator) result.decorator = {
		tagName: decorator.tagName,
		format: __assign({}, decorator.format)
	};
	return result;
}
function ensureParagraph(group, blockFormat, segmentFormat) {
	var lastBlock = group.blocks[group.blocks.length - 1];
	if ((lastBlock === null || lastBlock === void 0 ? void 0 : lastBlock.blockType) == "Paragraph") return mutateBlock(lastBlock);
	else {
		var paragraph = createParagraph(true, blockFormat, segmentFormat);
		addBlock(group, paragraph);
		return paragraph;
	}
}
function addSegment(group, newSegment, blockFormat, segmentFormat) {
	var paragraph = ensureParagraph(group, blockFormat, segmentFormat);
	var lastSegment = paragraph.segments[paragraph.segments.length - 1];
	if (blockFormat === null || blockFormat === void 0 ? void 0 : blockFormat.textIndent) {
		if (blockFormat.isTextIndentApplied && paragraph.segments.length == 0) delete paragraph.format.textIndent;
		else blockFormat.isTextIndentApplied = true;
		delete paragraph.format.isTextIndentApplied;
	}
	if (newSegment.segmentType == "SelectionMarker") {
		if (!lastSegment || !lastSegment.isSelected || !newSegment.isSelected) paragraph.segments.push(newSegment);
	} else {
		if (newSegment.isSelected && (lastSegment === null || lastSegment === void 0 ? void 0 : lastSegment.segmentType) == "SelectionMarker" && lastSegment.isSelected) paragraph.segments.pop();
		paragraph.segments.push(newSegment);
	}
	return paragraph;
}
function addLink(segment, link) {
	if (link.format.href || link.format.name) segment.link = {
		format: __assign({}, link.format),
		dataset: __assign({}, link.dataset)
	};
}
function addCode(segment, code) {
	if (code.format.fontFamily) segment.code = { format: __assign({}, code.format) };
}
function addDecorators(segment, context) {
	addLink(segment, context.link);
	addCode(segment, context.code);
}
function createSelectionMarker(format) {
	return {
		segmentType: "SelectionMarker",
		isSelected: true,
		format: __assign({}, format)
	};
}
function buildSelectionMarker(group, context, container, offset) {
	var lastPara = group.blocks[group.blocks.length - 1];
	var formatFromParagraph = !lastPara || lastPara.blockType != "Paragraph" ? {} : lastPara.decorator ? {
		fontFamily: lastPara.decorator.format.fontFamily,
		fontSize: lastPara.decorator.format.fontSize
	} : lastPara.segmentFormat ? {
		fontFamily: lastPara.segmentFormat.fontFamily,
		fontSize: lastPara.segmentFormat.fontSize
	} : {};
	var pendingFormat = context.pendingFormat && context.pendingFormat.insertPoint.node === container && context.pendingFormat.insertPoint.offset === offset ? context.pendingFormat.format : void 0;
	var marker = createSelectionMarker(Object.assign({}, context.defaultFormat, formatFromParagraph, context.segmentFormat, pendingFormat));
	addDecorators(marker, context);
	return marker;
}
function addSelectionMarker$1(group, context, container, offset) {
	var marker = buildSelectionMarker(group, context, container, offset);
	addSegment(group, marker, context.blockFormat, marker.format);
}
function getRegularSelectionOffsets(context, currentContainer) {
	var _a$5;
	var range = ((_a$5 = context.selection) === null || _a$5 === void 0 ? void 0 : _a$5.type) == "range" ? context.selection.range : null;
	return [(range === null || range === void 0 ? void 0 : range.startContainer) == currentContainer ? range.startOffset : -1, (range === null || range === void 0 ? void 0 : range.endContainer) == currentContainer ? range.endOffset : -1];
}
var childProcessor$1 = function(group, parent, context) {
	var _a$5 = __read(getRegularSelectionOffsets(context, parent), 2), nodeStartOffset = _a$5[0], nodeEndOffset = _a$5[1];
	var index$1 = 0;
	for (var child$1 = parent.firstChild; child$1; child$1 = child$1.nextSibling) {
		handleRegularSelection(index$1, context, group, nodeStartOffset, nodeEndOffset, parent);
		processChildNode(group, child$1, context);
		index$1++;
	}
	handleRegularSelection(index$1, context, group, nodeStartOffset, nodeEndOffset, parent);
};
function processChildNode(group, child$1, context) {
	if (isNodeOfType(child$1, "ELEMENT_NODE") && (child$1.style.display != "none" || context.processNonVisibleElements)) context.elementProcessors.element(group, child$1, context);
	else if (isNodeOfType(child$1, "TEXT_NODE")) context.elementProcessors["#text"](group, child$1, context);
}
function handleRegularSelection(index$1, context, group, nodeStartOffset, nodeEndOffset, container) {
	var _a$5;
	if (index$1 == nodeStartOffset) {
		context.isInSelection = true;
		addSelectionMarker$1(group, context, container, index$1);
	}
	if (index$1 == nodeEndOffset && ((_a$5 = context.selection) === null || _a$5 === void 0 ? void 0 : _a$5.type) == "range") {
		if (!context.selection.range.collapsed) addSelectionMarker$1(group, context, container, index$1);
		context.isInSelection = false;
	}
}
function createEntity(wrapper, isReadonly, segmentFormat, type, id) {
	if (isReadonly === void 0) isReadonly = true;
	return {
		segmentType: "Entity",
		blockType: "Entity",
		format: __assign({}, segmentFormat),
		entityFormat: {
			id,
			entityType: type,
			isReadonly
		},
		wrapper
	};
}
var blockElement = { display: "block" };
var defaultHTMLStyleMap = {
	address: blockElement,
	article: blockElement,
	aside: blockElement,
	b: { fontWeight: "bold" },
	blockquote: {
		display: "block",
		marginTop: "1em",
		marginBottom: "1em",
		marginLeft: "40px",
		marginRight: "40px"
	},
	br: blockElement,
	center: {
		display: "block",
		textAlign: "center"
	},
	dd: {
		display: "block",
		marginInlineStart: "40px"
	},
	div: blockElement,
	dl: {
		display: "block",
		marginTop: "1em",
		marginBottom: "1em"
	},
	dt: blockElement,
	em: { fontStyle: "italic" },
	fieldset: blockElement,
	figcaption: blockElement,
	figure: blockElement,
	footer: blockElement,
	form: blockElement,
	h1: {
		display: "block",
		fontWeight: "bold",
		fontSize: "2em"
	},
	h2: {
		display: "block",
		fontWeight: "bold",
		fontSize: "1.5em"
	},
	h3: {
		display: "block",
		fontWeight: "bold",
		fontSize: "1.17em"
	},
	h4: {
		display: "block",
		fontWeight: "bold"
	},
	h5: {
		display: "block",
		fontWeight: "bold",
		fontSize: "0.83em"
	},
	h6: {
		display: "block",
		fontWeight: "bold",
		fontSize: "0.67em"
	},
	header: blockElement,
	hr: blockElement,
	i: { fontStyle: "italic" },
	li: { display: "list-item" },
	main: blockElement,
	nav: blockElement,
	ol: __assign(__assign({}, blockElement), { paddingInlineStart: "40px" }),
	p: {
		display: "block",
		marginTop: "1em",
		marginBottom: "1em"
	},
	pre: {
		display: "block",
		fontFamily: "monospace",
		whiteSpace: "pre",
		marginTop: "1em",
		marginBottom: "1em"
	},
	s: { textDecoration: "line-through" },
	section: blockElement,
	strike: { textDecoration: "line-through" },
	strong: { fontWeight: "bold" },
	sub: {
		verticalAlign: "sub",
		fontSize: "smaller"
	},
	sup: {
		verticalAlign: "super",
		fontSize: "smaller"
	},
	table: {
		display: "table",
		boxSizing: "border-box"
	},
	td: { display: "table-cell" },
	th: {
		display: "table-cell",
		fontWeight: "bold"
	},
	u: { textDecoration: "underline" },
	ul: __assign(__assign({}, blockElement), { paddingInlineStart: "40px" })
};
function getDefaultStyle(element) {
	return defaultHTMLStyleMap[element.tagName.toLowerCase()] || {};
}
var BLOCK_DISPLAY_STYLES = [
	"block",
	"list-item",
	"table",
	"table-cell",
	"flex"
];
function isBlockElement(element) {
	var effectiveDisplay = (element.style.display == "none" ? null : element.style.display) || getDefaultStyle(element).display || "";
	return BLOCK_DISPLAY_STYLES.indexOf(effectiveDisplay) >= 0;
}
function parseFormat(element, parsers, format, context) {
	var defaultStyle = getDefaultStyle(element);
	parsers.forEach(function(parser) {
		parser === null || parser === void 0 || parser(format, element, context, defaultStyle);
	});
}
var SkippedStylesForBlockOnSegmentOnSegment = ["backgroundColor"];
var SkippedStylesForTable = [
	"marginLeft",
	"marginRight",
	"paddingLeft",
	"paddingRight"
];
function stackFormat$1(context, options, callback) {
	var segmentFormat = context.segmentFormat, blockFormat = context.blockFormat, linkFormat = context.link, codeFormat = context.code, decoratorFormat = context.blockDecorator;
	var segment = options.segment, paragraph = options.paragraph, link = options.link, code = options.code, blockDecorator = options.blockDecorator;
	try {
		context.segmentFormat = stackFormatInternal(segmentFormat, segment);
		context.blockFormat = stackFormatInternal(blockFormat, paragraph);
		context.link = stackLinkInternal(linkFormat, link);
		context.code = stackCodeInternal(codeFormat, code);
		context.blockDecorator = stackDecoratorInternal(decoratorFormat, blockDecorator);
		callback();
	} finally {
		context.segmentFormat = segmentFormat;
		context.blockFormat = blockFormat;
		context.link = linkFormat;
		context.code = codeFormat;
		context.blockDecorator = decoratorFormat;
	}
}
function stackLinkInternal(linkFormat, link) {
	switch (link) {
		case "linkDefault": return {
			format: { underline: true },
			dataset: {}
		};
		case "empty": return {
			format: {},
			dataset: {}
		};
		case "cloneFormat":
		default: return {
			dataset: linkFormat.dataset,
			format: __assign({}, linkFormat.format)
		};
	}
}
function stackCodeInternal(codeFormat, code) {
	switch (code) {
		case "codeDefault": return { format: { fontFamily: "monospace" } };
		case "empty": return { format: {} };
		default: return codeFormat;
	}
}
function stackDecoratorInternal(format, decorator) {
	switch (decorator) {
		case "empty": return {
			format: {},
			tagName: ""
		};
		default: return format;
	}
}
function stackFormatInternal(format, processType) {
	switch (processType) {
		case "empty": return {};
		case void 0: return format;
		default:
			var result_1 = __assign({}, format);
			getObjectKeys(format).forEach(function(key) {
				if (processType == "shallowCloneForBlock" && SkippedStylesForBlockOnSegmentOnSegment.indexOf(key) >= 0 || processType == "shallowCloneForGroup" && SkippedStylesForTable.indexOf(key) >= 0) delete result_1[key];
			});
			if (processType == "shallowClone" || processType == "shallowCloneForGroup") {
				var blockFormat = format;
				if (blockFormat.textIndent) {
					delete result_1.isTextIndentApplied;
					blockFormat.isTextIndentApplied = true;
				}
			}
			return result_1;
	}
}
var entityProcessor = function(group, element, context) {
	var isBlockEntity = isBlockElement(element) || element.style.display == "inline-block" && element.style.width == "100%";
	stackFormat$1(context, {
		segment: isBlockEntity ? "empty" : void 0,
		paragraph: "empty"
	}, function() {
		var _a$5;
		var entityModel = createEntity(element, true, context.segmentFormat);
		parseFormat(element, context.formatParsers.entity, entityModel.entityFormat, context);
		if (context.isInSelection) entityModel.isSelected = true;
		if (isBlockEntity) addBlock(group, entityModel);
		else {
			var paragraph = addSegment(group, entityModel);
			(_a$5 = context.domIndexer) === null || _a$5 === void 0 || _a$5.onSegment(element, paragraph, [entityModel]);
		}
	});
};
function createTableRow(format, height) {
	if (height === void 0) height = 0;
	return {
		height,
		format: __assign({}, format),
		cells: []
	};
}
function createTable(rowCount, format) {
	var rows = [];
	for (var i$1 = 0; i$1 < rowCount; i$1++) rows.push(createTableRow());
	return {
		blockType: "Table",
		rows,
		format: __assign({}, format),
		widths: [],
		dataset: {}
	};
}
function createTableCell(spanLeftOrColSpan, spanAboveOrRowSpan, isHeader, format, dataset) {
	var spanLeft = typeof spanLeftOrColSpan === "number" ? spanLeftOrColSpan > 1 : !!spanLeftOrColSpan;
	var spanAbove = typeof spanAboveOrRowSpan === "number" ? spanAboveOrRowSpan > 1 : !!spanAboveOrRowSpan;
	return {
		blockGroupType: "TableCell",
		blocks: [],
		format: __assign({}, format),
		spanLeft,
		spanAbove,
		isHeader: !!isHeader,
		dataset: __assign({}, dataset)
	};
}
function getBoundingClientRect(element) {
	return element.getBoundingClientRect();
}
function getSelectionRootNode(selection) {
	return !selection ? void 0 : selection.type == "range" ? selection.range.commonAncestorContainer : selection.type == "table" ? selection.table : selection.type == "image" ? selection.image : void 0;
}
function isElementOfType(element, tag) {
	var _a$5;
	return ((_a$5 = element === null || element === void 0 ? void 0 : element.tagName) === null || _a$5 === void 0 ? void 0 : _a$5.toLocaleLowerCase()) == tag;
}
var MarginValueRegex = /(-?\d+(\.\d+)?)([a-z]+|%)/;
var PixelPerInch = 96;
var DefaultRootFontSize$1 = 16;
function parseValueWithUnit(value, currentSizePxOrElement, resultUnit) {
	if (value === void 0) value = "";
	if (resultUnit === void 0) resultUnit = "px";
	var match = MarginValueRegex.exec(value);
	var result = 0;
	if (match) {
		var _a$5 = __read(match, 4);
		_a$5[0];
		var numStr = _a$5[1];
		_a$5[2];
		var unit = _a$5[3];
		var num = parseFloat(numStr);
		switch (unit) {
			case "px":
				result = num;
				break;
			case "pt":
				result = ptToPx(num);
				break;
			case "em":
				result = getFontSize$1(currentSizePxOrElement) * num;
				break;
			case "ex":
				result = getFontSize$1(currentSizePxOrElement) * num / 2;
				break;
			case "%":
				result = getFontSize$1(currentSizePxOrElement) * num / 100;
				break;
			case "in":
				result = num * PixelPerInch;
				break;
			case "rem":
				result = (getFontSize$1(currentSizePxOrElement) || DefaultRootFontSize$1) * num;
				break;
		}
	}
	if (result > 0 && resultUnit == "pt") result = pxToPt(result);
	return result;
}
function getFontSize$1(currentSizeOrElement) {
	var _a$5, _b$1;
	if (typeof currentSizeOrElement === "undefined") return 0;
	else if (typeof currentSizeOrElement === "number") return currentSizeOrElement;
	else {
		var styleInPt = (_b$1 = (_a$5 = currentSizeOrElement.ownerDocument.defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getComputedStyle(currentSizeOrElement).fontSize) !== null && _b$1 !== void 0 ? _b$1 : "";
		return ptToPx(parseFloat(styleInPt));
	}
}
function ptToPx(pt) {
	return Math.round(pt * 4e3 / 3) / 1e3;
}
function pxToPt(px) {
	return Math.round(px * 3e3 / 4) / 1e3;
}
var tableProcessor = function(group, tableElement, context) {
	stackFormat$1(context, {
		segment: "shallowCloneForBlock",
		paragraph: "shallowCloneForGroup"
	}, function() {
		var _a$5, _b$1;
		parseFormat(tableElement, context.formatParsers.block, context.blockFormat, context);
		var table = createTable(tableElement.rows.length, context.blockFormat);
		var tableSelection = ((_a$5 = context.selection) === null || _a$5 === void 0 ? void 0 : _a$5.type) == "table" ? context.selection : null;
		var hasTableSelection = (tableSelection === null || tableSelection === void 0 ? void 0 : tableSelection.table) == tableElement;
		var recalculateTableSize = shouldRecalculateTableSize(tableElement, context);
		if (context.allowCacheElement) table.cachedElement = tableElement;
		(_b$1 = context.domIndexer) === null || _b$1 === void 0 || _b$1.onTable(tableElement, table);
		parseFormat(tableElement, context.formatParsers.table, table.format, context);
		parseFormat(tableElement, context.formatParsers.tableBorder, table.format, context);
		parseFormat(tableElement, context.formatParsers.segmentOnBlock, context.segmentFormat, context);
		parseFormat(tableElement, context.formatParsers.dataset, table.dataset, context);
		addBlock(group, table);
		var columnPositions = [0];
		var hasColGroup = processColGroup(tableElement, context, columnPositions);
		var rowPositions = [0];
		var zoomScale = context.zoomScale || 1;
		var _loop_1 = function(row$1) {
			var tr = tableElement.rows[row$1];
			var tableRow = table.rows[row$1];
			var tbody = tr.parentNode;
			if (isNodeOfType(tbody, "ELEMENT_NODE") && (isElementOfType(tbody, "tbody") || isElementOfType(tbody, "thead") || isElementOfType(tbody, "tfoot"))) parseFormat(tbody, context.formatParsers.tableRow, tableRow.format, context);
			else if (context.allowCacheElement) tableRow.cachedElement = tr;
			parseFormat(tr, context.formatParsers.tableRow, tableRow.format, context);
			stackFormat$1(context, {
				paragraph: "shallowClone",
				segment: "shallowClone"
			}, function() {
				var parent = tr.parentElement;
				var isInTableSection = parent && getIsInTableSection(parent);
				if (isInTableSection) {
					parseFormat(parent, context.formatParsers.block, context.blockFormat, context);
					parseFormat(parent, context.formatParsers.segmentOnBlock, context.segmentFormat, context);
				}
				parseFormat(tr, context.formatParsers.block, context.blockFormat, context);
				parseFormat(tr, context.formatParsers.segmentOnBlock, context.segmentFormat, context);
				tableRow.height = parseInt(tr.style.height) || 0;
				var _loop_2 = function(sourceCol$1, targetCol$1) {
					for (; tableRow.cells[targetCol$1]; targetCol$1++);
					var td = tr.cells[sourceCol$1];
					var hasSelectionBeforeCell = context.isInSelection;
					if (recalculateTableSize) {
						var colEnd = targetCol$1 + td.colSpan;
						var rowEnd = row$1 + td.rowSpan;
						var needCalcWidth = columnPositions[colEnd] === void 0;
						var needCalcHeight = rowPositions[rowEnd] === void 0;
						if (needCalcWidth || needCalcHeight) {
							var rect = getBoundingClientRect(td);
							if (rect.width > 0 || rect.height > 0) {
								if (needCalcWidth) {
									var pos = columnPositions[targetCol$1];
									columnPositions[colEnd] = (typeof pos == "number" ? pos : 0) + rect.width / zoomScale;
								}
								if (needCalcHeight) rowPositions[rowEnd] = rowPositions[row$1] + rect.height / zoomScale;
							}
						}
					}
					stackFormat$1(context, {
						paragraph: "shallowClone",
						segment: "shallowClone"
					}, function() {
						parseFormat(td, context.formatParsers.block, context.blockFormat, context);
						parseFormat(td, context.formatParsers.segmentOnTableCell, context.segmentFormat, context);
						var cellFormat = __assign({}, context.blockFormat);
						var dataset = {};
						parseFormat(td, context.formatParsers.tableCell, cellFormat, context);
						parseFormat(td, context.formatParsers.tableBorder, cellFormat, context);
						parseFormat(td, context.formatParsers.dataset, dataset, context);
						for (var colSpan = 1; colSpan <= (td.colSpan == 0 ? 1 : td.colSpan); colSpan++, targetCol$1++) for (var rowSpan = 1; rowSpan <= (td.rowSpan == 0 ? isInTableSection ? translateRowSpanZero(parent, td) : 1 : td.rowSpan); rowSpan++) {
							var hasTd = colSpan == 1 && rowSpan == 1;
							var cell = createTableCell(colSpan > 1, rowSpan > 1, td.tagName == "TH", cellFormat);
							cell.dataset = __assign({}, dataset);
							var spannedRow = table.rows[row$1 + rowSpan - 1];
							if (spannedRow) spannedRow.cells[targetCol$1] = cell;
							if (hasTd) {
								if (context.allowCacheElement && !hasColGroup) cell.cachedElement = td;
								var _a$6 = context.listFormat, listParent = _a$6.listParent, levels = _a$6.levels;
								context.listFormat.listParent = void 0;
								context.listFormat.levels = [];
								try {
									context.elementProcessors.child(cell, td, context);
								} finally {
									context.listFormat.listParent = listParent;
									context.listFormat.levels = levels;
								}
							}
							var hasSelectionAfterCell = context.isInSelection;
							if (hasSelectionBeforeCell && hasSelectionAfterCell || hasTableSelection && tableSelection && row$1 >= tableSelection.firstRow && row$1 <= tableSelection.lastRow && targetCol$1 >= tableSelection.firstColumn && targetCol$1 <= tableSelection.lastColumn) cell.isSelected = true;
						}
					});
					out_targetCol_1 = targetCol$1;
				};
				var out_targetCol_1;
				for (var sourceCol = 0, targetCol = 0; sourceCol < tr.cells.length; sourceCol++) {
					_loop_2(sourceCol, targetCol);
					targetCol = out_targetCol_1;
				}
			});
			for (var col = 0; col < tableRow.cells.length; col++) if (!tableRow.cells[col]) tableRow.cells[col] = createTableCell(false, false, false, context.blockFormat);
		};
		for (var row = 0; row < tableElement.rows.length; row++) _loop_1(row);
		table.widths = calcSizes(columnPositions);
		var heights = calcSizes(rowPositions);
		table.rows.forEach(function(row$1, i$1) {
			if (heights[i$1] > 0) row$1.height = heights[i$1];
		});
	});
};
function translateRowSpanZero(parent, td) {
	var amountOfRows = parent.rows.length;
	var tdIndex = -1;
	for (var i$1 = 0; i$1 < parent.rows.length; i$1++) {
		var row = parent.rows[i$1];
		for (var j$1 = 0; j$1 < row.cells.length; j$1++) if (row.cells[j$1] === td) {
			tdIndex = i$1;
			break;
		}
		if (tdIndex !== -1) break;
	}
	return amountOfRows - tdIndex;
}
function calcSizes(positions) {
	var result = [];
	var lastPos = 0;
	for (var i$1 = positions.length - 1; i$1 >= 0; i$1--) {
		var pos = positions[i$1];
		if (typeof pos == "number") {
			lastPos = pos;
			break;
		}
	}
	for (var i$1 = positions.length - 2; i$1 >= 0; i$1--) {
		var pos = positions[i$1];
		if (pos === void 0) result[i$1] = 0;
		else {
			result[i$1] = lastPos - pos;
			lastPos = pos;
		}
	}
	return result;
}
function processColGroup(table, context, result) {
	var _a$5, _b$1;
	var lastPos = 0;
	var hasColGroup = false;
	for (var child$1 = table.firstChild; child$1; child$1 = child$1.nextSibling) if (isNodeOfType(child$1, "ELEMENT_NODE") && child$1.tagName == "COLGROUP") {
		hasColGroup = true;
		for (var col = child$1.firstChild; col; col = col.nextSibling) if (isNodeOfType(col, "ELEMENT_NODE") && col.tagName == "COL") {
			var colFormat = {};
			parseFormat(col, context.formatParsers.tableColumn, colFormat, context);
			for (var i$1 = 0; i$1 < parseInt((_a$5 = col.getAttribute("span")) !== null && _a$5 !== void 0 ? _a$5 : "1"); i$1++) if (colFormat.width === void 0) result.push(void 0);
			else {
				var width = parseValueWithUnit((_b$1 = colFormat.width) !== null && _b$1 !== void 0 ? _b$1 : "", void 0, "px");
				result.push(width + lastPos);
				lastPos += width;
			}
		}
	}
	return hasColGroup;
}
function shouldRecalculateTableSize(table, context) {
	switch (context.recalculateTableSize) {
		case true:
		case "all": return true;
		case "selected":
			var selectionRoot = getSelectionRootNode(context.selection);
			return !!selectionRoot && (selectionRoot == table || table.contains(selectionRoot) || selectionRoot.contains(table));
		default: return false;
	}
}
function getIsInTableSection(element) {
	return isElementOfType(element, "tbody") || isElementOfType(element, "thead") || isElementOfType(element, "tfoot");
}
function validate(input, def) {
	var result = false;
	if (def.isOptional && typeof input === "undefined" || def.allowNull && input === null) result = true;
	else if (!def.isOptional && typeof input === "undefined" || !def.allowNull && input === null) return false;
	else switch (def.type) {
		case "string":
			result = typeof input === "string" && (typeof def.value === "undefined" || input === def.value);
			break;
		case "number":
			result = typeof input === "number" && (typeof def.value === "undefined" || areSameNumbers(def.value, input)) && (typeof def.minValue === "undefined" || input >= def.minValue) && (typeof def.maxValue === "undefined" || input <= def.maxValue);
			break;
		case "boolean":
			result = typeof input === "boolean" && (typeof def.value === "undefined" || input === def.value);
			break;
		case "array":
			result = Array.isArray(input) && (typeof def.minLength === "undefined" || input.length >= def.minLength) && (typeof def.maxLength === "undefined" || input.length <= def.maxLength) && input.every(function(x$1) {
				return validate(x$1, def.itemDef);
			});
			break;
		case "object":
			result = typeof input === "object" && getObjectKeys(def.propertyDef).every(function(x$1) {
				return validate(input[x$1], def.propertyDef[x$1]);
			});
			break;
	}
	return result;
}
function areSameNumbers(n1, n2) {
	return Math.abs(n1 - n2) < .001;
}
var EditingInfoDatasetName = "editingInfo";
function getMetadata(model, definition) {
	var metadataString = model.dataset[EditingInfoDatasetName];
	var obj = null;
	try {
		obj = JSON.parse(metadataString);
	} catch (_a$5) {}
	return !definition || validate(obj, definition) ? obj : null;
}
function updateMetadata(model, callback, definition) {
	var obj = getMetadata(model, definition);
	if (callback) {
		obj = callback(obj);
		if (!obj) delete model.dataset[EditingInfoDatasetName];
		else if (!definition || validate(obj, definition)) model.dataset[EditingInfoDatasetName] = JSON.stringify(obj);
	}
	return obj;
}
function hasMetadata(model) {
	return !!model.dataset[EditingInfoDatasetName];
}
var StartsWithUnsupportedCharacter = /^[.|\-|_|\d]/;
function getSafeIdSelector(id) {
	if (!id) return id;
	if (id.match(StartsWithUnsupportedCharacter)) return "[id=\"" + id + "\"]";
	else return "#" + id;
}
function moveChildNodes(target, source, keepExistingChildren) {
	if (!target) return;
	while (!keepExistingChildren && target.firstChild) target.removeChild(target.firstChild);
	while (source === null || source === void 0 ? void 0 : source.firstChild) target.appendChild(source.firstChild);
}
function wrapAllChildNodes(parent, tagName) {
	var newElement = parent.ownerDocument.createElement(tagName);
	moveChildNodes(newElement, parent);
	parent.appendChild(newElement);
	return newElement;
}
function wrap(doc, node, wrapperTag) {
	var _a$5;
	var wrapper = doc.createElement(wrapperTag);
	(_a$5 = node.parentNode) === null || _a$5 === void 0 || _a$5.insertBefore(wrapper, node);
	wrapper.appendChild(node);
	return wrapper;
}
function unwrap$1(node) {
	var parentNode = node ? node.parentNode : null;
	if (!parentNode) return null;
	while (node.firstChild) parentNode.insertBefore(node.firstChild, node);
	parentNode.removeChild(node);
	return parentNode;
}
function applyFormat(element, appliers, format, context) {
	appliers.forEach(function(applier) {
		applier === null || applier === void 0 || applier(format, element, context);
	});
}
var ENTITY_INFO_NAME = "_Entity";
var ENTITY_INFO_SELECTOR = "." + ENTITY_INFO_NAME;
var ENTITY_TYPE_PREFIX = "_EType_";
var ENTITY_ID_PREFIX = "_EId_";
var ENTITY_READONLY_PREFIX = "_EReadonly_";
var ZERO_WIDTH_SPACE = "​";
var DELIMITER_BEFORE = "entityDelimiterBefore";
var DELIMITER_AFTER = "entityDelimiterAfter";
var BLOCK_ENTITY_CONTAINER = "_E_EBlockEntityContainer";
var BLOCK_ENTITY_CONTAINER_SELECTOR = "." + BLOCK_ENTITY_CONTAINER;
function isEntityElement(node) {
	return isNodeOfType(node, "ELEMENT_NODE") && node.classList.contains(ENTITY_INFO_NAME);
}
function findClosestEntityWrapper(startNode, domHelper) {
	return domHelper.findClosestElementAncestor(startNode, ENTITY_INFO_SELECTOR);
}
function findClosestBlockEntityContainer(node, domHelper) {
	return domHelper.findClosestElementAncestor(node, BLOCK_ENTITY_CONTAINER_SELECTOR);
}
function getAllEntityWrappers(root$12) {
	return toArray(root$12.querySelectorAll("." + ENTITY_INFO_NAME));
}
function parseEntityFormat(wrapper) {
	var isEntity = false;
	var format = {};
	wrapper.classList.forEach(function(name) {
		isEntity = parseEntityClassName(name, format) || isEntity;
	});
	if (!isEntity) {
		format.isFakeEntity = true;
		format.isReadonly = !wrapper.isContentEditable;
	}
	return format;
}
function parseEntityClassName(className, format) {
	if (className == ENTITY_INFO_NAME) return true;
	else if (className.indexOf(ENTITY_TYPE_PREFIX) == 0) format.entityType = className.substring(ENTITY_TYPE_PREFIX.length);
	else if (className.indexOf(ENTITY_ID_PREFIX) == 0) format.id = className.substring(ENTITY_ID_PREFIX.length);
	else if (className.indexOf(ENTITY_READONLY_PREFIX) == 0) format.isReadonly = className.substring(ENTITY_READONLY_PREFIX.length) == "1";
}
function generateEntityClassNames(format) {
	var _a$5;
	return format.isFakeEntity ? "" : ENTITY_INFO_NAME + " " + ENTITY_TYPE_PREFIX + ((_a$5 = format.entityType) !== null && _a$5 !== void 0 ? _a$5 : "") + " " + (format.id ? "" + ENTITY_ID_PREFIX + format.id + " " : "") + ENTITY_READONLY_PREFIX + (format.isReadonly ? "1" : "0");
}
function isEntityDelimiter(element, isBefore) {
	var matchBefore = isBefore === void 0 || isBefore;
	var matchAfter = isBefore === void 0 || !isBefore;
	return isElementOfType(element, "span") && (matchAfter && element.classList.contains(DELIMITER_AFTER) || matchBefore && element.classList.contains(DELIMITER_BEFORE)) && element.textContent === ZERO_WIDTH_SPACE;
}
function isBlockEntityContainer(element) {
	return isElementOfType(element, "div") && element.classList.contains(BLOCK_ENTITY_CONTAINER);
}
function addDelimiters(doc, element, format, context) {
	var _a$5 = __read(getDelimiters(element), 2), delimiterAfter = _a$5[0], delimiterBefore = _a$5[1];
	if (!delimiterAfter) {
		delimiterAfter = insertDelimiter(doc, element, true);
		if (context && format) applyFormat(delimiterAfter, context.formatAppliers.segment, format, context);
	}
	if (!delimiterBefore) {
		delimiterBefore = insertDelimiter(doc, element, false);
		if (context && format) applyFormat(delimiterBefore, context.formatAppliers.segment, format, context);
	}
	return [delimiterAfter, delimiterBefore];
}
function getDelimiters(entityWrapper) {
	var result = [];
	var nextElementSibling = entityWrapper.nextElementSibling, previousElementSibling = entityWrapper.previousElementSibling;
	result.push(isDelimiter(nextElementSibling, DELIMITER_AFTER), isDelimiter(previousElementSibling, DELIMITER_BEFORE));
	return result;
}
function isDelimiter(el, className) {
	return (el === null || el === void 0 ? void 0 : el.classList.contains(className)) && el.textContent == ZERO_WIDTH_SPACE ? el : void 0;
}
function insertDelimiter(doc, element, isAfter) {
	var _a$5;
	var span = doc.createElement("span");
	span.className = isAfter ? DELIMITER_AFTER : DELIMITER_BEFORE;
	span.appendChild(doc.createTextNode(ZERO_WIDTH_SPACE));
	(_a$5 = element.parentNode) === null || _a$5 === void 0 || _a$5.insertBefore(span, isAfter ? element.nextSibling : element);
	return span;
}
function reuseCachedElement(parent, element, refNode, context) {
	var _a$5;
	if (element.parentNode == parent) {
		var isEntity = isEntityElement(element);
		while (refNode && refNode != element && (isEntity || !isEntityElement(refNode))) {
			var next$1 = refNode.nextSibling;
			(_a$5 = refNode.parentNode) === null || _a$5 === void 0 || _a$5.removeChild(refNode);
			if (isNodeOfType(refNode, "ELEMENT_NODE")) context === null || context === void 0 || context.removedBlockElements.push(refNode);
			refNode = next$1;
		}
		if (refNode && refNode == element) refNode = refNode.nextSibling;
		else parent.insertBefore(element, refNode);
	} else parent.insertBefore(element, refNode);
	return refNode;
}
function normalizeRect(clientRect) {
	var _a$5 = clientRect || {
		left: 0,
		right: 0,
		top: 0,
		bottom: 0
	}, left = _a$5.left, right = _a$5.right, top = _a$5.top, bottom = _a$5.bottom;
	return left === 0 && right === 0 && top === 0 && bottom === 0 ? null : {
		left: Math.round(left),
		right: Math.round(right),
		top: Math.round(top),
		bottom: Math.round(bottom)
	};
}
function scrollRectIntoView(scrollContainer, visibleRect, domHelper, targetRect, scrollMargin, preferTop) {
	if (scrollMargin === void 0) scrollMargin = 0;
	if (preferTop === void 0) preferTop = false;
	var zoomScale;
	var margin = 0;
	if (scrollMargin != 0) {
		zoomScale = getZoomScale(domHelper, zoomScale);
		margin = Math.max(0, Math.min(scrollMargin * zoomScale, (visibleRect.bottom - visibleRect.top - targetRect.bottom + targetRect.top) / 2));
	}
	var top = targetRect.top - margin;
	var bottom = targetRect.bottom + margin;
	var height = bottom - top;
	var scrollUp = function() {
		zoomScale = getZoomScale(domHelper, zoomScale);
		scrollContainer.scrollTop -= (visibleRect.top - top) / zoomScale;
	};
	var scrollDown = function() {
		zoomScale = getZoomScale(domHelper, zoomScale);
		scrollContainer.scrollTop += (bottom - visibleRect.bottom) / zoomScale;
	};
	var needsScrollUp = top < visibleRect.top;
	var needsScrollDown = bottom > visibleRect.bottom;
	if (height > visibleRect.bottom - visibleRect.top) if (preferTop) scrollUp();
	else scrollDown();
	else if (preferTop) {
		if (needsScrollUp) scrollUp();
		else if (needsScrollDown) scrollDown();
	} else if (needsScrollDown) scrollDown();
	else if (needsScrollUp) scrollUp();
}
function getZoomScale(domHelper, knownZoomScale) {
	if (knownZoomScale === void 0) return domHelper.calculateZoomScale();
	else return knownZoomScale;
}
function getHiddenProperty(node, key) {
	var hiddenProperty = node.__roosterjsHiddenProperty;
	return hiddenProperty ? hiddenProperty[key] : void 0;
}
function setHiddenProperty(node, key, value) {
	var nodeWithHiddenProperty = node;
	var hiddenProperty = nodeWithHiddenProperty.__roosterjsHiddenProperty || {};
	hiddenProperty[key] = value;
	nodeWithHiddenProperty.__roosterjsHiddenProperty = hiddenProperty;
}
var UndeletableLinkKey = "undeletable";
function setLinkUndeletable(a$1, undeletable) {
	setHiddenProperty(a$1, UndeletableLinkKey, undeletable);
}
function isLinkUndeletable(a$1) {
	return !!getHiddenProperty(a$1, UndeletableLinkKey);
}
function createListLevel(listType, format, dataset) {
	return {
		listType,
		format: __assign({}, format),
		dataset: __assign({}, dataset)
	};
}
function createListItem(levels, format) {
	var formatHolder = createSelectionMarker(format);
	formatHolder.isSelected = false;
	return {
		blockType: "BlockGroup",
		blockGroupType: "ListItem",
		blocks: [],
		levels: levels ? levels.map(function(level) {
			return createListLevel(level.listType, level.format, level.dataset);
		}) : [],
		formatHolder,
		format: {}
	};
}
function createFormatContainer(tag, format) {
	return {
		blockType: "BlockGroup",
		blockGroupType: "FormatContainer",
		tagName: tag,
		blocks: [],
		format: __assign({}, format)
	};
}
function createText(text, format, link, code) {
	var result = {
		segmentType: "Text",
		text,
		format: __assign({}, format)
	};
	if (link) addLink(result, link);
	if (code) addCode(result, code);
	return result;
}
function createImage(src, format) {
	return {
		segmentType: "Image",
		src,
		format: __assign({}, format),
		dataset: {}
	};
}
function createParagraphDecorator(tagName, format) {
	return {
		tagName: tagName.toLocaleLowerCase(),
		format: __assign({}, format)
	};
}
function createGeneralSegment(element, format) {
	return {
		blockType: "BlockGroup",
		blockGroupType: "General",
		segmentType: "General",
		format: __assign({}, format),
		blocks: [],
		element
	};
}
function createGeneralBlock(element) {
	return {
		blockType: "BlockGroup",
		blockGroupType: "General",
		element,
		blocks: [],
		format: {}
	};
}
function createDivider(tagName, format) {
	return {
		blockType: "Divider",
		tagName,
		format: __assign({}, format)
	};
}
function createEmptyModel(format) {
	var model = createContentModelDocument(format);
	var paragraph = createParagraph(false, void 0, format);
	paragraph.segments.push(createSelectionMarker(format), createBr(format));
	model.blocks.push(paragraph);
	return model;
}
function addTextSegment(group, text, context) {
	var _a$5;
	var textModel;
	if (text) {
		var paragraph = ensureParagraph(group, context.blockFormat);
		if (!hasSpacesOnly(text) || ((_a$5 = paragraph === null || paragraph === void 0 ? void 0 : paragraph.segments.length) !== null && _a$5 !== void 0 ? _a$5 : 0) > 0 || isWhiteSpacePreserved(paragraph === null || paragraph === void 0 ? void 0 : paragraph.format.whiteSpace)) {
			textModel = createText(text, context.segmentFormat);
			if (context.isInSelection) textModel.isSelected = true;
			addDecorators(textModel, context);
			addSegment(group, textModel, context.blockFormat);
		}
	}
	return textModel;
}
function isGeneralSegment(group) {
	return group.blockGroupType == "General" && group.segmentType == "General";
}
function mergeTextSegments(block) {
	var lastText = null;
	for (var i$1 = 0; i$1 < block.segments.length; i$1++) {
		var segment = block.segments[i$1];
		if (segment.segmentType != "Text") lastText = null;
		else if (!lastText || !segmentsWithSameFormat(lastText, segment)) lastText = segment;
		else {
			lastText.text += segment.text;
			block.segments.splice(i$1, 1);
			i$1--;
		}
	}
}
function segmentsWithSameFormat(seg1, seg2) {
	return !!seg1.isSelected == !!seg2.isSelected && areSameFormats(seg1.format, seg2.format) && areSameLinks(seg1.link, seg2.link) && areSameCodes(seg1.code, seg2.code);
}
function areSameLinks(link1, link2) {
	return !link1 && !link2 || link1 && link2 && areSameFormats(link1.format, link2.format) && areSameFormats(link1.dataset, link2.dataset);
}
function areSameCodes(code1, code2) {
	return !code1 && !code2 || code1 && code2 && areSameFormats(code1.format, code2.format);
}
var defaultContentModelFormatMap = {
	a: {
		underline: true,
		textColor: void 0
	},
	blockquote: {
		marginTop: "1em",
		marginBottom: "1em",
		marginLeft: "40px",
		marginRight: "40px"
	},
	code: { fontFamily: "monospace" },
	dd: { marginLeft: "40px" },
	dl: {
		marginTop: "1em",
		marginBottom: "1em"
	},
	h1: {
		fontWeight: "bold",
		fontSize: "2em"
	},
	h2: {
		fontWeight: "bold",
		fontSize: "1.5em"
	},
	h3: {
		fontWeight: "bold",
		fontSize: "1.17em"
	},
	h4: {
		fontWeight: "bold",
		fontSize: "1em"
	},
	h5: {
		fontWeight: "bold",
		fontSize: "0.83em"
	},
	h6: {
		fontWeight: "bold",
		fontSize: "0.67em"
	},
	p: {
		marginTop: "1em",
		marginBottom: "1em"
	},
	pre: {
		fontFamily: "monospace",
		whiteSpace: "pre",
		marginTop: "1em",
		marginBottom: "1em"
	},
	th: { fontWeight: "bold" }
};
var handleBlock = function(doc, parent, block, context, refNode) {
	var handlers = context.modelHandlers;
	switch (block.blockType) {
		case "Table":
			refNode = handlers.table(doc, parent, block, context, refNode);
			break;
		case "Paragraph":
			refNode = handlers.paragraph(doc, parent, block, context, refNode);
			break;
		case "Entity":
			refNode = handlers.entityBlock(doc, parent, block, context, refNode);
			break;
		case "Divider":
			refNode = handlers.divider(doc, parent, block, context, refNode);
			break;
		case "BlockGroup":
			switch (block.blockGroupType) {
				case "General":
					refNode = handlers.generalBlock(doc, parent, block, context, refNode);
					break;
				case "FormatContainer":
					refNode = handlers.formatContainer(doc, parent, block, context, refNode);
					break;
				case "ListItem":
					refNode = handlers.listItem(doc, parent, block, context, refNode);
					break;
			}
			break;
	}
	return refNode;
};
var handleBlockGroupChildren = function(doc, parent, group, context) {
	var _a$5;
	var listFormat = context.listFormat;
	var nodeStack = listFormat.nodeStack;
	var refNode = parent.firstChild;
	try {
		group.blocks.forEach(function(childBlock, index$1) {
			var _a$6;
			if (index$1 == 0 || childBlock.blockType != "BlockGroup" || childBlock.blockGroupType != "ListItem") listFormat.nodeStack = [];
			refNode = context.modelHandlers.block(doc, parent, childBlock, context, refNode);
			if (childBlock.blockType == "Entity") (_a$6 = context.domIndexer) === null || _a$6 === void 0 || _a$6.onBlockEntity(childBlock, group);
		});
		while (refNode) {
			var next$1 = refNode.nextSibling;
			if (isNodeOfType(refNode, "ELEMENT_NODE")) context.rewriteFromModel.removedBlockElements.push(refNode);
			(_a$5 = refNode.parentNode) === null || _a$5 === void 0 || _a$5.removeChild(refNode);
			refNode = next$1;
		}
	} finally {
		listFormat.nodeStack = nodeStack;
	}
};
function handleSegmentCommon(doc, segmentNode, containerNode, segment, context, segmentNodes) {
	var _a$5;
	if (!segmentNode.firstChild) context.regularSelection.current.segment = segmentNode;
	applyFormat(containerNode, context.formatAppliers.styleBasedSegment, segment.format, context);
	segmentNodes === null || segmentNodes === void 0 || segmentNodes.push(segmentNode);
	context.modelHandlers.segmentDecorator(doc, containerNode, segment, context, segmentNodes);
	applyFormat(containerNode, context.formatAppliers.elementBasedSegment, segment.format, context);
	(_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, segment, segmentNode);
}
var handleBr = function(doc, parent, segment, context, segmentNodes) {
	var br = doc.createElement("br");
	var element = doc.createElement("span");
	element.appendChild(br);
	parent.appendChild(element);
	handleSegmentCommon(doc, br, element, segment, context, segmentNodes);
};
var handleDivider = function(doc, parent, divider, context, refNode) {
	var _a$5;
	var element = context.allowCacheElement ? divider.cachedElement : void 0;
	if (element && !divider.isSelected) refNode = reuseCachedElement(parent, element, refNode, context.rewriteFromModel);
	else {
		element = doc.createElement(divider.tagName);
		if (context.allowCacheElement) divider.cachedElement = element;
		parent.insertBefore(element, refNode);
		context.rewriteFromModel.addedBlockElements.push(element);
		applyFormat(element, context.formatAppliers.divider, divider.format, context);
		if (divider.size) element.setAttribute("size", divider.size);
	}
	(_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, divider, element);
	return refNode;
};
var BlockEntityContainer = "_E_EBlockEntityContainer";
var handleEntityBlock = function(doc, parent, entityModel, context, refNode) {
	var _a$5, _b$1;
	var entityFormat = entityModel.entityFormat, wrapper = entityModel.wrapper;
	applyFormat(wrapper, context.formatAppliers.entity, entityFormat, context);
	var isCursorAroundEntity = context.addDelimiterForEntity && wrapper.style.display == "inline-block" && wrapper.style.width == "100%";
	var isContained = (_a$5 = wrapper.parentElement) === null || _a$5 === void 0 ? void 0 : _a$5.classList.contains(BlockEntityContainer);
	refNode = reuseCachedElement(parent, isContained && isCursorAroundEntity ? wrapper.parentElement : wrapper, refNode, context.rewriteFromModel);
	if (isCursorAroundEntity) {
		if (!isContained) wrap(doc, wrapper, "div").classList.add(BlockEntityContainer);
		addDelimiters(doc, wrapper, getSegmentFormat(context), context);
	}
	(_b$1 = context.onNodeCreated) === null || _b$1 === void 0 || _b$1.call(context, entityModel, wrapper);
	return refNode;
};
var handleEntitySegment = function(doc, parent, entityModel, context, newSegments) {
	var _a$5;
	var entityFormat = entityModel.entityFormat, wrapper = entityModel.wrapper, format = entityModel.format;
	parent.appendChild(wrapper);
	newSegments === null || newSegments === void 0 || newSegments.push(wrapper);
	if (getObjectKeys(format).length > 0) applyFormat(wrap(doc, wrapper, "span"), context.formatAppliers.segment, format, context);
	applyFormat(wrapper, context.formatAppliers.entity, entityFormat, context);
	if (context.addDelimiterForEntity && entityFormat.isReadonly) {
		var _b$1 = __read(addDelimiters(doc, wrapper, getSegmentFormat(context), context), 2), after = _b$1[0], before = _b$1[1];
		if (newSegments) {
			newSegments.push(after, before);
			if (after.firstChild) newSegments.push(after.firstChild);
			if (before.firstChild) newSegments.push(before.firstChild);
		}
		context.regularSelection.current.segment = after;
	} else context.regularSelection.current.segment = wrapper;
	(_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, entityModel, wrapper);
};
function getSegmentFormat(context) {
	var _a$5;
	return __assign(__assign({}, (_a$5 = context.pendingFormat) === null || _a$5 === void 0 ? void 0 : _a$5.format), context.defaultFormat);
}
function stackFormat(context, tagNameOrFormat, callback) {
	var newFormat = typeof tagNameOrFormat === "string" ? context.defaultContentModelFormatMap[tagNameOrFormat] : tagNameOrFormat;
	if (newFormat) {
		var implicitFormat = context.implicitFormat;
		try {
			context.implicitFormat = __assign(__assign({}, implicitFormat), newFormat);
			callback();
		} finally {
			context.implicitFormat = implicitFormat;
		}
	} else callback();
}
var PreChildFormat = {
	fontFamily: "monospace",
	whiteSpace: "pre"
};
var handleFormatContainer = function(doc, parent, container, context, refNode) {
	var _a$5;
	var element = context.allowCacheElement ? container.cachedElement : void 0;
	if (element) {
		refNode = reuseCachedElement(parent, element, refNode, context.rewriteFromModel);
		context.modelHandlers.blockGroupChildren(doc, element, container, context);
	} else if (!isBlockGroupEmpty(container)) {
		var containerNode_1 = doc.createElement(container.tagName);
		if (context.allowCacheElement) container.cachedElement = containerNode_1;
		parent.insertBefore(containerNode_1, refNode);
		context.rewriteFromModel.addedBlockElements.push(containerNode_1);
		stackFormat(context, container.tagName, function() {
			applyFormat(containerNode_1, context.formatAppliers.container, container.format, context);
			applyFormat(containerNode_1, context.formatAppliers.segmentOnBlock, container.format, context);
			applyFormat(containerNode_1, context.formatAppliers.container, container.format, context);
		});
		if (container.tagName == "pre") stackFormat(context, PreChildFormat, function() {
			context.modelHandlers.blockGroupChildren(doc, containerNode_1, container, context);
		});
		else context.modelHandlers.blockGroupChildren(doc, containerNode_1, container, context);
		element = containerNode_1;
	}
	if (element) (_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, container, element);
	return refNode;
};
var handleGeneralBlock = function(doc, parent, group, context, refNode) {
	var _a$5;
	var node = group.element;
	if (refNode && node.parentNode == parent) refNode = reuseCachedElement(parent, node, refNode, context.rewriteFromModel);
	else {
		node = node.cloneNode();
		group.element = node;
		applyFormat(node, context.formatAppliers.general, group.format, context);
		parent.insertBefore(node, refNode);
		context.rewriteFromModel.addedBlockElements.push(node);
	}
	(_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, group, node);
	context.modelHandlers.blockGroupChildren(doc, node, group, context);
	return refNode;
};
var handleGeneralSegment = function(doc, parent, group, context, segmentNodes) {
	var _a$5;
	var node = group.element.cloneNode();
	group.element = node;
	parent.appendChild(node);
	if (isNodeOfType(node, "ELEMENT_NODE")) {
		handleSegmentCommon(doc, node, wrap(doc, node, "span"), group, context, segmentNodes);
		applyFormat(node, context.formatAppliers.general, group.format, context);
		(_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, group, node);
	}
	context.modelHandlers.blockGroupChildren(doc, node, group, context);
};
var handleImage = function(doc, parent, imageModel, context, segmentNodes) {
	var img = doc.createElement("img");
	var element = document.createElement("span");
	parent.appendChild(element);
	element.appendChild(img);
	img.src = imageModel.src;
	if (imageModel.alt) img.alt = imageModel.alt;
	if (imageModel.title) img.title = imageModel.title;
	applyFormat(img, context.formatAppliers.image, imageModel.format, context);
	applyFormat(img, context.formatAppliers.dataset, imageModel.dataset, context);
	var _a$5 = imageModel.format, width = _a$5.width, height = _a$5.height;
	var widthNum = width ? parseValueWithUnit(width) : 0;
	var heightNum = height ? parseValueWithUnit(height) : 0;
	if (widthNum > 0) img.width = widthNum;
	if (heightNum > 0) img.height = heightNum;
	if (imageModel.isSelectedAsImageSelection) context.imageSelection = {
		type: "image",
		image: img
	};
	handleSegmentCommon(doc, img, element, imageModel, context, segmentNodes);
};
function applyMetadata(model, applier, format, context) {
	if (applier) updateMetadata(model, function(metadata) {
		applier.applierFunction(metadata, format, context);
		return metadata;
	}, applier.metadataDefinition);
}
var handleList = function(doc, parent, listItem, context, refNode) {
	var _a$5, _b$1, _c$1;
	var layer = 0;
	var nodeStack = context.listFormat.nodeStack;
	if (nodeStack.length == 0) nodeStack.push({ node: parent });
	for (; layer < listItem.levels.length && layer + 1 < nodeStack.length; layer++) {
		var stackLevel = nodeStack[layer + 1];
		var itemLevel = listItem.levels[layer];
		if (stackLevel.listType != itemLevel.listType || ((_a$5 = stackLevel.dataset) === null || _a$5 === void 0 ? void 0 : _a$5.editingInfo) != itemLevel.dataset.editingInfo || itemLevel.listType == "OL" && typeof itemLevel.format.startNumberOverride === "number" || itemLevel.listType == "UL" && itemLevel.format.listStyleType != ((_b$1 = stackLevel.format) === null || _b$1 === void 0 ? void 0 : _b$1.listStyleType)) break;
	}
	nodeStack.splice(layer + 1);
	for (; layer < listItem.levels.length; layer++) {
		var level = listItem.levels[layer];
		var newList = doc.createElement(level.listType || "UL");
		nodeStack[nodeStack.length - 1].node.insertBefore(newList, layer == 0 ? refNode : null);
		nodeStack.push(__assign({ node: newList }, level));
		applyFormat(newList, context.formatAppliers.listLevelThread, level.format, context);
		applyMetadata(level, context.metadataAppliers.listLevel, level.format, context);
		applyFormat(newList, context.formatAppliers.listLevel, level.format, context);
		applyFormat(newList, context.formatAppliers.dataset, level.dataset, context);
		(_c$1 = context.onNodeCreated) === null || _c$1 === void 0 || _c$1.call(context, level, newList);
	}
	return refNode;
};
var genericRoleElements = new Set([
	"div",
	"span",
	"p",
	"section",
	"article",
	"aside",
	"header",
	"footer",
	"main",
	"nav",
	"address",
	"blockquote",
	"pre",
	"figure",
	"figcaption",
	"hgroup"
]);
function isGenericRoleElement(element) {
	if (!element) return false;
	var tagName = element.tagName.toLowerCase();
	return genericRoleElements.has(tagName);
}
var HtmlRoleAttribute = "role";
var PresentationRoleValue = "presentation";
var handleListItem = function(doc, parent, listItem, context, refNode) {
	var _a$5, _b$1;
	refNode = context.modelHandlers.list(doc, parent, listItem, context, refNode);
	var nodeStack = context.listFormat.nodeStack;
	var listParent = ((_a$5 = nodeStack === null || nodeStack === void 0 ? void 0 : nodeStack[(nodeStack === null || nodeStack === void 0 ? void 0 : nodeStack.length) - 1]) === null || _a$5 === void 0 ? void 0 : _a$5.node) || parent;
	var li = doc.createElement("li");
	var level = listItem.levels[listItem.levels.length - 1];
	listParent.insertBefore(li, (refNode === null || refNode === void 0 ? void 0 : refNode.parentNode) == listParent ? refNode : null);
	context.rewriteFromModel.addedBlockElements.push(li);
	if (level) {
		applyFormat(li, context.formatAppliers.segment, listItem.formatHolder.format, context);
		applyFormat(li, context.formatAppliers.listItemThread, level.format, context);
		applyMetadata(level, context.metadataAppliers.listItem, listItem.format, context);
		applyFormat(li, context.formatAppliers.listItemElement, listItem.format, context);
		stackFormat(context, listItem.formatHolder.format, function() {
			context.modelHandlers.blockGroupChildren(doc, li, listItem, context);
		});
	} else {
		listItem.blocks.forEach(setParagraphNotImplicit);
		context.modelHandlers.blockGroupChildren(doc, li, listItem, context);
		unwrap$1(li);
	}
	for (var index$1 = 0; index$1 < li.children.length; index$1++) {
		var element = li.children.item(index$1);
		if (isGenericRoleElement(element)) element.setAttribute(HtmlRoleAttribute, PresentationRoleValue);
	}
	(_b$1 = context.onNodeCreated) === null || _b$1 === void 0 || _b$1.call(context, listItem, li);
	return refNode;
};
var OptimizeTags = [
	"SPAN",
	"B",
	"EM",
	"I",
	"U",
	"SUB",
	"SUP",
	"STRIKE",
	"S",
	"A",
	"CODE"
];
function mergeNode(root$12) {
	for (var child$1 = root$12.firstChild; child$1;) {
		var next$1 = child$1.nextSibling;
		if (next$1 && isNodeOfType(child$1, "ELEMENT_NODE") && isNodeOfType(next$1, "ELEMENT_NODE") && child$1.tagName == next$1.tagName && OptimizeTags.indexOf(child$1.tagName) >= 0 && hasSameAttributes(child$1, next$1)) {
			while (next$1.firstChild) child$1.appendChild(next$1.firstChild);
			next$1.parentNode.removeChild(next$1);
		} else child$1 = next$1;
	}
}
function hasSameAttributes(element1, element2) {
	var attr1 = element1.attributes;
	var attr2 = element2.attributes;
	if (attr1.length != attr2.length) return false;
	for (var i$1 = 0; i$1 < attr1.length; i$1++) if (attr1[i$1].name != attr2[i$1].name || attr1[i$1].value != attr2[i$1].value) return false;
	return true;
}
function removeUnnecessarySpan(root$12) {
	for (var child$1 = root$12.firstChild; child$1;) if (isNodeOfType(child$1, "ELEMENT_NODE") && child$1.tagName == "SPAN" && child$1.attributes.length == 0) {
		var node = child$1;
		var refNode = child$1.nextSibling;
		child$1 = child$1.nextSibling;
		while (node.lastChild) {
			var newNode = node.lastChild;
			root$12.insertBefore(newNode, refNode);
			refNode = newNode;
		}
		root$12.removeChild(node);
	} else child$1 = child$1.nextSibling;
}
function optimize(root$12, context) {
	if (isEntityElement(root$12)) return;
	removeUnnecessarySpan(root$12);
	mergeNode(root$12);
	for (var child$1 = root$12.firstChild; child$1; child$1 = child$1.nextSibling) optimize(child$1, context);
	normalizeTextNode(root$12, context);
}
function normalizeTextNode(root$12, context) {
	var _a$5, _b$1, _c$1, _d$1;
	var lastText = null;
	var child$1;
	var next$1;
	var selection = context.regularSelection;
	for (child$1 = root$12.firstChild, next$1 = child$1 ? child$1.nextSibling : null; child$1; child$1 = next$1, next$1 = child$1 ? child$1.nextSibling : null) if (!isNodeOfType(child$1, "TEXT_NODE")) lastText = null;
	else if (!lastText) lastText = child$1;
	else {
		var originalLength = (_b$1 = (_a$5 = lastText.nodeValue) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0;
		(_c$1 = context.domIndexer) === null || _c$1 === void 0 || _c$1.onMergeText(lastText, child$1);
		lastText.nodeValue += (_d$1 = child$1.nodeValue) !== null && _d$1 !== void 0 ? _d$1 : "";
		if (selection) {
			updateSelection(selection.start, lastText, child$1, originalLength);
			updateSelection(selection.end, lastText, child$1, originalLength);
		}
		root$12.removeChild(child$1);
	}
}
function updateSelection(mark, lastText, nextText, lastTextOriginalLength) {
	if (mark && mark.offset == void 0) {
		if (mark.segment == lastText) mark.offset = lastTextOriginalLength;
		else if (mark.segment == nextText) mark.segment = lastText;
	}
}
var DefaultParagraphTag = "div";
var handleParagraph = function(doc, parent, paragraph, context, refNode) {
	var _a$5;
	var container = context.allowCacheElement ? paragraph.cachedElement : void 0;
	if (container && paragraph.segments.every(function(x$1) {
		return x$1.segmentType != "General" && !x$1.isSelected;
	})) refNode = reuseCachedElement(parent, container, refNode, context.rewriteFromModel);
	else stackFormat(context, ((_a$5 = paragraph.decorator) === null || _a$5 === void 0 ? void 0 : _a$5.tagName) || null, function() {
		var _a$6, _b$1, _c$1, _d$1, _e$1;
		var needParagraphWrapper = !paragraph.isImplicit || !!paragraph.decorator || getObjectKeys(paragraph.format).length > 0 && paragraph.segments.some(function(segment) {
			return segment.segmentType != "SelectionMarker";
		});
		var formatOnWrapper = needParagraphWrapper ? __assign(__assign({}, ((_a$6 = paragraph.decorator) === null || _a$6 === void 0 ? void 0 : _a$6.format) || {}), paragraph.segmentFormat) : {};
		container = doc.createElement(((_b$1 = paragraph.decorator) === null || _b$1 === void 0 ? void 0 : _b$1.tagName) || DefaultParagraphTag);
		parent.insertBefore(container, refNode);
		context.regularSelection.current = {
			block: needParagraphWrapper ? container : container.parentNode,
			segment: null
		};
		var handleSegments = function() {
			var parent$1 = container;
			if (parent$1) {
				var firstSegment = paragraph.segments[0];
				if ((firstSegment === null || firstSegment === void 0 ? void 0 : firstSegment.segmentType) == "SelectionMarker") context.modelHandlers.text(doc, parent$1, __assign(__assign({}, firstSegment), {
					segmentType: "Text",
					text: ""
				}), context, []);
				paragraph.segments.forEach(function(segment) {
					var newSegments = [];
					context.modelHandlers.segment(doc, parent$1, segment, context, newSegments);
					newSegments.forEach(function(node) {
						var _a$7;
						(_a$7 = context.domIndexer) === null || _a$7 === void 0 || _a$7.onSegment(node, paragraph, [segment]);
					});
				});
			}
		};
		if (needParagraphWrapper) {
			stackFormat(context, formatOnWrapper, handleSegments);
			applyFormat(container, context.formatAppliers.block, paragraph.format, context);
			applyFormat(container, context.formatAppliers.container, paragraph.format, context);
			applyFormat(container, context.formatAppliers.segmentOnBlock, formatOnWrapper, context);
			(_c$1 = context.paragraphMap) === null || _c$1 === void 0 || _c$1.applyMarkerToDom(container, paragraph);
		} else handleSegments();
		optimize(container, context);
		refNode = container.nextSibling;
		if (container) {
			(_d$1 = context.onNodeCreated) === null || _d$1 === void 0 || _d$1.call(context, paragraph, container);
			(_e$1 = context.domIndexer) === null || _e$1 === void 0 || _e$1.onParagraph(container);
		}
		if (needParagraphWrapper) {
			if (context.allowCacheElement) paragraph.cachedElement = container;
			context.rewriteFromModel.addedBlockElements.push(container);
		} else {
			unwrap$1(container);
			container = void 0;
		}
	});
	return refNode;
};
var handleSegment = function(doc, parent, segment, context, segmentNodes) {
	var regularSelection = context.regularSelection;
	if (segment.isSelected && !regularSelection.start) regularSelection.start = __assign({}, regularSelection.current);
	switch (segment.segmentType) {
		case "Text":
			context.modelHandlers.text(doc, parent, segment, context, segmentNodes);
			break;
		case "Br":
			context.modelHandlers.br(doc, parent, segment, context, segmentNodes);
			break;
		case "Image":
			context.modelHandlers.image(doc, parent, segment, context, segmentNodes);
			break;
		case "General":
			context.modelHandlers.generalSegment(doc, parent, segment, context, segmentNodes);
			break;
		case "Entity":
			context.modelHandlers.entitySegment(doc, parent, segment, context, segmentNodes);
			break;
	}
	if (segment.isSelected && regularSelection.start) regularSelection.end = __assign({}, regularSelection.current);
};
var handleSegmentDecorator = function(_, parent, segment, context) {
	var code = segment.code, link = segment.link;
	if (isNodeOfType(parent, "ELEMENT_NODE")) {
		if (link) stackFormat(context, "a", function() {
			var _a$5;
			var a$1 = wrapAllChildNodes(parent, "a");
			applyFormat(a$1, context.formatAppliers.link, link.format, context);
			applyFormat(a$1, context.formatAppliers.dataset, link.dataset, context);
			(_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, link, a$1);
		});
		if (code) stackFormat(context, "code", function() {
			var _a$5;
			var codeNode = wrapAllChildNodes(parent, "code");
			applyFormat(codeNode, context.formatAppliers.code, code.format, context);
			(_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, code, codeNode);
		});
	}
};
var handleTable = function(doc, parent, table, context, refNode) {
	var _a$5, _b$1, _c$1, _d$1, _e$1, _f$1, _g;
	if (isBlockEmpty(table)) return refNode;
	var tableNode = context.allowCacheElement ? table.cachedElement : void 0;
	if (tableNode) {
		refNode = reuseCachedElement(parent, tableNode, refNode, context.rewriteFromModel);
		moveChildNodes(tableNode);
	} else {
		tableNode = doc.createElement("table");
		if (context.allowCacheElement) table.cachedElement = tableNode;
		parent.insertBefore(tableNode, refNode);
		context.rewriteFromModel.addedBlockElements.push(tableNode);
		applyFormat(tableNode, context.formatAppliers.block, table.format, context);
		applyFormat(tableNode, context.formatAppliers.table, table.format, context);
		applyFormat(tableNode, context.formatAppliers.tableBorder, table.format, context);
		applyFormat(tableNode, context.formatAppliers.dataset, table.dataset, context);
	}
	(_a$5 = context.onNodeCreated) === null || _a$5 === void 0 || _a$5.call(context, table, tableNode);
	var tbody = doc.createElement("tbody");
	tableNode.appendChild(tbody);
	for (var row = 0; row < table.rows.length; row++) {
		var tableRow = table.rows[row];
		if (tableRow.cells.length == 0) continue;
		var tr = context.allowCacheElement && tableRow.cachedElement || doc.createElement("tr");
		tbody.appendChild(tr);
		moveChildNodes(tr);
		if (!tableRow.cachedElement) {
			if (context.allowCacheElement) tableRow.cachedElement = tr;
			applyFormat(tr, context.formatAppliers.tableRow, tableRow.format, context);
		}
		(_b$1 = context.onNodeCreated) === null || _b$1 === void 0 || _b$1.call(context, tableRow, tr);
		var _loop_1 = function(col$1) {
			var cell = tableRow.cells[col$1];
			if (cell.isSelected) {
				var tableSelection = context.tableSelection || {
					type: "table",
					table: tableNode,
					firstColumn: col$1,
					lastColumn: col$1,
					firstRow: row,
					lastRow: row
				};
				if (tableSelection.table == tableNode) {
					tableSelection.lastColumn = Math.max(tableSelection.lastColumn, col$1);
					tableSelection.lastRow = Math.max(tableSelection.lastRow, row);
				}
				context.tableSelection = tableSelection;
			}
			if (!cell.spanAbove && !cell.spanLeft) {
				var tag = cell.isHeader ? "th" : "td";
				var td_1 = context.allowCacheElement && cell.cachedElement || doc.createElement(tag);
				tr.appendChild(td_1);
				var rowSpan = 1;
				var colSpan = 1;
				var width = table.widths[col$1];
				var height = tableRow.height;
				for (; (_d$1 = (_c$1 = table.rows[row + rowSpan]) === null || _c$1 === void 0 ? void 0 : _c$1.cells[col$1]) === null || _d$1 === void 0 ? void 0 : _d$1.spanAbove; rowSpan++) height += table.rows[row + rowSpan].height;
				for (; (_e$1 = tableRow.cells[col$1 + colSpan]) === null || _e$1 === void 0 ? void 0 : _e$1.spanLeft; colSpan++) width += table.widths[col$1 + colSpan];
				if (rowSpan > 1) td_1.rowSpan = rowSpan;
				if (colSpan > 1) td_1.colSpan = colSpan;
				if (!cell.cachedElement || cell.format.useBorderBox && hasMetadata(table)) {
					if (width > 0 && !td_1.style.width) td_1.style.width = width + "px";
					if (height > 0 && !td_1.style.height) td_1.style.height = height + "px";
				}
				stackFormat(context, tag, function() {
					if (!cell.cachedElement) {
						if (context.allowCacheElement) cell.cachedElement = td_1;
						applyFormat(td_1, context.formatAppliers.block, cell.format, context);
						applyFormat(td_1, context.formatAppliers.tableCell, cell.format, context);
						applyFormat(td_1, context.formatAppliers.tableCellBorder, cell.format, context);
						applyFormat(td_1, context.formatAppliers.dataset, cell.dataset, context);
					}
					context.modelHandlers.blockGroupChildren(doc, td_1, cell, context);
				});
				(_f$1 = context.onNodeCreated) === null || _f$1 === void 0 || _f$1.call(context, cell, td_1);
			}
		};
		for (var col = 0; col < tableRow.cells.length; col++) _loop_1(col);
	}
	(_g = context.domIndexer) === null || _g === void 0 || _g.onTable(tableNode, table);
	return refNode;
};
var handleText = function(doc, parent, segment, context, segmentNodes) {
	var txt = doc.createTextNode(segment.text);
	var element = doc.createElement("span");
	parent.appendChild(element);
	element.appendChild(txt);
	context.formatAppliers.text.forEach(function(applier) {
		return applier(segment.format, txt, context);
	});
	handleSegmentCommon(doc, txt, element, segment, context, segmentNodes);
};
var defaultContentModelHandlers = {
	block: handleBlock,
	blockGroupChildren: handleBlockGroupChildren,
	br: handleBr,
	entityBlock: handleEntityBlock,
	entitySegment: handleEntitySegment,
	generalBlock: handleGeneralBlock,
	generalSegment: handleGeneralSegment,
	divider: handleDivider,
	image: handleImage,
	list: handleList,
	listItem: handleListItem,
	paragraph: handleParagraph,
	formatContainer: handleFormatContainer,
	segment: handleSegment,
	segmentDecorator: handleSegmentDecorator,
	table: handleTable,
	text: handleText
};
var ariaFormatHandler = {
	parse: function(format, element) {
		var ariaDescribedBy = element.getAttribute("aria-describedby");
		var title = element.getAttribute("title");
		if (ariaDescribedBy) format.ariaDescribedBy = ariaDescribedBy;
		if (title) format.title = title;
	},
	apply: function(format, element) {
		if (format.ariaDescribedBy) element.setAttribute("aria-describedby", format.ariaDescribedBy);
		if (format.title) element.setAttribute("title", format.title);
	}
};
var DeprecatedColors = [
	"inactiveborder",
	"activeborder",
	"inactivecaptiontext",
	"inactivecaption",
	"activecaption",
	"appworkspace",
	"infobackground",
	"background",
	"buttonhighlight",
	"buttonshadow",
	"captiontext",
	"infotext",
	"menutext",
	"menu",
	"scrollbar",
	"threeddarkshadow",
	"threedface",
	"threedhighlight",
	"threedlightshadow",
	"threedfhadow",
	"windowtext",
	"windowframe",
	"window"
];
var BlackColor$1 = "rgb(0, 0, 0)";
var HEX3_REGEX = /^#([a-fA-F0-9])([a-fA-F0-9])([a-fA-F0-9])$/;
var HEX6_REGEX = /^#([a-fA-F0-9]{2})([a-fA-F0-9]{2})([a-fA-F0-9]{2})$/;
var RGB_REGEX = /^rgb\(\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*\)$/;
var RGBA_REGEX = /^rgba\(\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*,\s*(\d+(?:\.\d+)?)\s*\)$/;
var VARIABLE_REGEX$1 = /^\s*var\(\s*(\-\-[a-zA-Z0-9\-_]+)\s*(?:,\s*(.*))?\)\s*$/;
var VARIABLE_PREFIX$1 = "var(";
var VARIABLE_POSTFIX = ")";
var COLOR_VAR_PREFIX = "--darkColor";
function getColor(element, isBackground, isDarkMode, darkColorHandler, fallback) {
	var color = (isBackground ? element.style.backgroundColor : element.style.color) || element.getAttribute(isBackground ? "bgcolor" : "color") || fallback;
	if (color && DeprecatedColors.indexOf(color) > -1) color = isBackground ? void 0 : BlackColor$1;
	else if (darkColorHandler && color) {
		var match = color.startsWith(VARIABLE_PREFIX$1) ? VARIABLE_REGEX$1.exec(color) : null;
		if (match) color = match[2] || "";
		else if (isDarkMode) color = findLightColorFromDarkColor(color, darkColorHandler.knownColors) || "";
	}
	return color;
}
function setColor(element, color, isBackground, isDarkMode, darkColorHandler) {
	var _a$5, _b$1;
	var match = color && color.startsWith(VARIABLE_PREFIX$1) ? VARIABLE_REGEX$1.exec(color) : null, _c$1 = __read(match !== null && match !== void 0 ? match : [], 3);
	_c$1[0];
	var existingKey = _c$1[1], fallbackColor = _c$1[2];
	color = fallbackColor !== null && fallbackColor !== void 0 ? fallbackColor : color;
	if (darkColorHandler && color) {
		var colorType = isBackground ? "background" : "text";
		var key = existingKey || darkColorHandler.generateColorKey(color, void 0, colorType, element);
		var darkModeColor = ((_b$1 = (_a$5 = darkColorHandler.knownColors) === null || _a$5 === void 0 ? void 0 : _a$5[key]) === null || _b$1 === void 0 ? void 0 : _b$1.darkModeColor) || darkColorHandler.getDarkColor(color, void 0, colorType, element);
		darkColorHandler.updateKnownColor(isDarkMode, key, {
			lightModeColor: color,
			darkModeColor
		});
		color = isDarkMode ? "" + VARIABLE_PREFIX$1 + key + ", " + color + VARIABLE_POSTFIX : color;
	}
	element.removeAttribute(isBackground ? "bgcolor" : "color");
	element.style.setProperty(isBackground ? "background-color" : "color", color || null);
}
var defaultGenerateColorKey = function(lightColor) {
	return COLOR_VAR_PREFIX + "_" + lightColor.replace(/[^\d\w]/g, "_");
};
function parseColor(color) {
	color = (color || "").trim();
	var match;
	if (match = color.match(HEX3_REGEX)) return [
		parseInt(match[1] + match[1], 16),
		parseInt(match[2] + match[2], 16),
		parseInt(match[3] + match[3], 16)
	];
	else if (match = color.match(HEX6_REGEX)) return [
		parseInt(match[1], 16),
		parseInt(match[2], 16),
		parseInt(match[3], 16)
	];
	else if (match = color.match(RGB_REGEX) || color.match(RGBA_REGEX)) return [
		parseInt(match[1]),
		parseInt(match[2]),
		parseInt(match[3])
	];
	else return null;
}
function findLightColorFromDarkColor(darkColor, knownColors) {
	var rgbSearch = parseColor(darkColor);
	if (rgbSearch && knownColors) {
		var key = getObjectKeys(knownColors).find(function(key$1) {
			var rgbCurrent = parseColor(knownColors[key$1].darkModeColor);
			return rgbCurrent && rgbCurrent[0] == rgbSearch[0] && rgbCurrent[1] == rgbSearch[1] && rgbCurrent[2] == rgbSearch[2];
		});
		if (key) return knownColors[key].lightModeColor;
	}
	return null;
}
function shouldSetValue(value, normalValue, existingValue, defaultValue) {
	return !!value && value != "inherit" && !!(value != normalValue || existingValue || defaultValue && value != defaultValue);
}
var backgroundColorFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		var backgroundColor = getColor(element, true, !!context.isDarkMode, context.darkColorHandler) || defaultStyle.backgroundColor;
		if (shouldSetValue(backgroundColor, "transparent", void 0, defaultStyle.backgroundColor)) format.backgroundColor = backgroundColor;
	},
	apply: function(format, element, context) {
		if (format.backgroundColor) setColor(element, format.backgroundColor, true, !!context.isDarkMode, context.darkColorHandler);
	}
};
var boldFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		var fontWeight = element.style.fontWeight || defaultStyle.fontWeight;
		if (shouldSetValue(fontWeight, "400", format.fontWeight, defaultStyle.fontWeight)) format.fontWeight = fontWeight;
	},
	apply: function(format, element, context) {
		if (typeof format.fontWeight === "undefined") return;
		var blockFontWeight = context.implicitFormat.fontWeight;
		if (blockFontWeight && blockFontWeight != format.fontWeight || !blockFontWeight && format.fontWeight && format.fontWeight != "normal") if (format.fontWeight == "bold") wrapAllChildNodes(element, "b");
		else element.style.fontWeight = format.fontWeight || "normal";
	}
};
var borderBoxFormatHandler = {
	parse: function(format, element) {
		var _a$5;
		if (((_a$5 = element.style) === null || _a$5 === void 0 ? void 0 : _a$5.boxSizing) == "border-box") format.useBorderBox = true;
	},
	apply: function(format, element) {
		if (format.useBorderBox) element.style.boxSizing = "border-box";
	}
};
var BorderKeys = [
	"borderTop",
	"borderRight",
	"borderBottom",
	"borderLeft"
];
var BorderWidthKeys = [
	"borderTopWidth",
	"borderRightWidth",
	"borderBottomWidth",
	"borderLeftWidth"
];
var BorderRadiusKeys = [
	"borderTopLeftRadius",
	"borderTopRightRadius",
	"borderBottomLeftRadius",
	"borderBottomRightRadius"
];
var AllKeys = BorderKeys.concat(BorderRadiusKeys);
var borderFormatHandler = {
	parse: function(format, element, _, defaultStyle) {
		BorderKeys.forEach(function(key, i$1) {
			var _a$5;
			var value = element.style[key];
			var defaultWidth = (_a$5 = defaultStyle[BorderWidthKeys[i$1]]) !== null && _a$5 !== void 0 ? _a$5 : "0px";
			var width = element.style[BorderWidthKeys[i$1]];
			if (width == "0") width = "0px";
			if (value && width != defaultWidth) format[key] = value == "none" ? "" : value;
		});
		var borderRadius = element.style.borderRadius;
		if (borderRadius) format.borderRadius = borderRadius;
		else BorderRadiusKeys.forEach(function(key) {
			var value = element.style[key];
			if (value) format[key] = value;
		});
	},
	apply: function(format, element) {
		AllKeys.forEach(function(key) {
			var value = format[key];
			if (value) element.style[key] = value;
		});
		if (format.borderRadius) element.style.borderRadius = format.borderRadius;
	}
};
var boxShadowFormatHandler = {
	parse: function(format, element) {
		var _a$5;
		if ((_a$5 = element.style) === null || _a$5 === void 0 ? void 0 : _a$5.boxShadow) format.boxShadow = element.style.boxShadow;
	},
	apply: function(format, element) {
		if (format.boxShadow) element.style.boxShadow = format.boxShadow;
	}
};
var datasetFormatHandler = {
	parse: function(format, element) {
		var dataset = element.dataset;
		getObjectKeys(dataset).forEach(function(key) {
			format[key] = dataset[key] || "";
		});
	},
	apply: function(format, element) {
		getObjectKeys(format).forEach(function(key) {
			element.dataset[key] = format[key];
		});
	}
};
var directionFormatHandler = {
	parse: function(format, element, _, defaultStyle) {
		var dir = element.style.direction || element.dir || defaultStyle.direction;
		if (dir) format.direction = dir == "rtl" ? "rtl" : "ltr";
	},
	apply: function(format, element) {
		if (format.direction) element.style.direction = format.direction;
		if (format.direction == "rtl" && isElementOfType(element, "table")) element.style.justifySelf = "flex-end";
	}
};
var displayFormatHandler = {
	parse: function(format, element) {
		var display = element.style.display;
		if (display) format.display = display;
	},
	apply: function(format, element) {
		if (format.display) element.style.display = format.display;
	}
};
var entityFormatHandler = {
	parse: function(format, element) {
		Object.assign(format, parseEntityFormat(element));
	},
	apply: function(format, element) {
		if (!format.isFakeEntity) element.className = generateEntityClassNames(format);
		if (format.isReadonly) element.contentEditable = "false";
		else element.removeAttribute("contenteditable");
	}
};
var floatFormatHandler = {
	parse: function(format, element) {
		var float = element.style.float || element.getAttribute("align");
		if (float) format.float = float;
	},
	apply: function(format, element) {
		if (format.float) element.style.float = format.float;
	}
};
var fontFamilyFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		var fontFamily = element.style.fontFamily || defaultStyle.fontFamily;
		if (fontFamily && fontFamily != "inherit") format.fontFamily = fontFamily;
	},
	apply: function(format, element, context) {
		if (format.fontFamily && format.fontFamily != context.implicitFormat.fontFamily) element.style.fontFamily = format.fontFamily;
	}
};
var superOrSubScriptFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		var verticalAlign = element.style.verticalAlign || defaultStyle.verticalAlign;
		if (isSuperOrSubScript(element.style.fontSize || defaultStyle.fontSize, verticalAlign)) format.superOrSubScriptSequence = (format.superOrSubScriptSequence || "").split(" ").concat(verticalAlign).join(" ").trim();
	},
	apply: function(format, element) {
		if (format.superOrSubScriptSequence) format.superOrSubScriptSequence.split(" ").reverse().forEach(function(value) {
			var tagName = value == "super" ? "sup" : value == "sub" ? "sub" : null;
			if (tagName) wrapAllChildNodes(element, tagName);
		});
	}
};
function isSuperOrSubScript(fontSize, verticalAlign) {
	return fontSize == "smaller" && (verticalAlign == "sub" || verticalAlign == "super");
}
var fontSizeFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		var fontSize = element.style.fontSize || defaultStyle.fontSize;
		var verticalAlign = element.style.verticalAlign || defaultStyle.verticalAlign;
		if (fontSize && !isSuperOrSubScript(fontSize, verticalAlign) && fontSize != "inherit") {
			if (element.style.fontSize) format.fontSize = normalizeFontSize(fontSize, context.segmentFormat.fontSize, context);
			else if (defaultStyle.fontSize) format.fontSize = fontSize;
		}
	},
	apply: function(format, element, context) {
		if (format.fontSize && format.fontSize != context.implicitFormat.fontSize) element.style.fontSize = format.fontSize;
	}
};
var KnownFontSizes = {
	"xx-small": "6.75pt",
	"x-small": "7.5pt",
	small: "9.75pt",
	medium: "12pt",
	large: "13.5pt",
	"x-large": "18pt",
	"xx-large": "24pt",
	"xxx-large": "36pt"
};
function normalizeFontSize(fontSize, contextFont, context) {
	var knownFontSize = KnownFontSizes[fontSize];
	var isRemUnit = fontSize.endsWith("rem");
	if (knownFontSize) return knownFontSize;
	else if (fontSize == "smaller" || fontSize == "larger" || fontSize.endsWith("em") || fontSize.endsWith("%") || isRemUnit) if (!contextFont && !isRemUnit) return;
	else {
		var existingFontSize = isRemUnit ? context.rootFontSize : parseValueWithUnit(contextFont);
		if (existingFontSize) switch (fontSize) {
			case "smaller": return Math.round(existingFontSize * 500 / 6) / 100 + "px";
			case "larger": return Math.round(existingFontSize * 600 / 5) / 100 + "px";
			default: return parseValueWithUnit(fontSize, existingFontSize, "px") + "px";
		}
	}
	else if (fontSize == "inherit" || fontSize == "revert" || fontSize == "unset") return;
	else return fontSize;
}
var ResultMap$1 = {
	start: {
		ltr: "left",
		rtl: "right"
	},
	center: {
		ltr: "center",
		rtl: "center"
	},
	end: {
		ltr: "right",
		rtl: "left"
	},
	initial: {
		ltr: "initial",
		rtl: "initial"
	},
	justify: {
		ltr: "justify",
		rtl: "justify"
	}
};
function calcAlign(align, dir) {
	switch (align) {
		case "center": return "center";
		case "left": return dir == "rtl" ? "end" : "start";
		case "right": return dir == "rtl" ? "start" : "end";
		case "start":
		case "end": return align;
		case "justify":
		case "initial": return align;
		default: return;
	}
}
var htmlAlignFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		if (!element.style.textAlign) {
			directionFormatHandler.parse(format, element, context, defaultStyle);
			var htmlAlign = element.getAttribute("align");
			if (htmlAlign) {
				format.htmlAlign = calcAlign(htmlAlign, format.direction);
				delete format.textAlign;
				delete context.blockFormat.textAlign;
			}
		}
	},
	apply: function(format, element) {
		var dir = format.direction == "rtl" ? "rtl" : "ltr";
		if (format.htmlAlign) element.setAttribute("align", ResultMap$1[format.htmlAlign][dir]);
	}
};
var idFormatHandler = {
	parse: function(format, element) {
		if (element.id) format.id = element.id;
	},
	apply: function(format, element) {
		if (format.id) element.id = format.id;
	}
};
function getImageState(element) {
	return getHiddenProperty(element, "imageState");
}
function setImageState(element, marker) {
	setHiddenProperty(element, "imageState", marker);
}
var imageStateFormatHandler = {
	parse: function(format, element) {
		var marker = getImageState(element);
		if (marker) format.imageState = marker;
	},
	apply: function(format, element) {
		if (format.imageState) setImageState(element, format.imageState);
	}
};
var italicFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		var fontStyle = element.style.fontStyle || defaultStyle.fontStyle;
		if (fontStyle == "italic" || fontStyle == "oblique") format.italic = true;
		else if (fontStyle == "initial" || fontStyle == "normal") format.italic = false;
	},
	apply: function(format, element, context) {
		if (typeof format.italic === "undefined") return;
		if (!!context.implicitFormat.italic != !!format.italic) if (format.italic) wrapAllChildNodes(element, "i");
		else element.style.fontStyle = "normal";
	}
};
var letterSpacingFormatHandler = {
	parse: function(format, element, _, defaultStyle) {
		var letterSpacing = element.style.letterSpacing || defaultStyle.letterSpacing;
		if (shouldSetValue(letterSpacing, "normal", format.letterSpacing, defaultStyle.letterSpacing)) format.letterSpacing = letterSpacing;
	},
	apply: function(format, element) {
		if (format.letterSpacing) element.style.letterSpacing = format.letterSpacing;
	}
};
var lineHeightFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		var lineHeight = element.style.lineHeight || defaultStyle.lineHeight;
		if (lineHeight && lineHeight != "inherit") format.lineHeight = lineHeight;
	},
	apply: function(format, element) {
		if (format.lineHeight) element.style.lineHeight = format.lineHeight;
	}
};
var linkFormatHandler = {
	parse: function(format, element) {
		if (isElementOfType(element, "a")) {
			var name_1 = element.name;
			var href = element.getAttribute("href");
			var target = element.target;
			var rel = element.rel;
			var id = element.id;
			var className = element.className;
			var title = element.title;
			if (name_1) format.name = name_1;
			if (href) format.href = href;
			if (target) format.target = target;
			if (id) format.anchorId = id;
			if (rel) format.relationship = rel;
			if (title) format.anchorTitle = title;
			if (className) format.anchorClass = className;
		}
	},
	apply: function(format, element) {
		if (isElementOfType(element, "a") && (format.href || format.name)) {
			if (format.href) element.href = format.href;
			if (format.name) element.name = format.name;
			if (format.target) element.target = format.target;
			if (format.anchorId) element.id = format.anchorId;
			if (format.anchorClass) element.className = format.anchorClass;
			if (format.anchorTitle) element.title = format.anchorTitle;
			if (format.relationship) element.rel = format.relationship;
		}
	}
};
var listItemThreadFormatHandler = {
	parse: function(format, element, context, defaultStyles) {
		var listFormat = context.listFormat;
		var depth = listFormat.levels.length;
		var display = element.style.display || defaultStyles.display;
		if (display && display != "list-item") format.displayForDummyItem = display;
		else if (isLiUnderOl(element) && depth > 0) {
			listFormat.threadItemCounts[depth - 1]++;
			listFormat.threadItemCounts.splice(depth);
			listFormat.levels.forEach(function(level) {
				delete level.format.startNumberOverride;
			});
		}
	},
	apply: function(format, element, context) {
		var _a$5;
		if (format.displayForDummyItem) element.style.display = format.displayForDummyItem;
		else if (isLiUnderOl(element)) {
			var listFormat = context.listFormat;
			var threadItemCounts = listFormat.threadItemCounts;
			var index$1 = listFormat.nodeStack.length - 2;
			if (index$1 >= 0) {
				threadItemCounts.splice(index$1 + 1);
				threadItemCounts[index$1] = ((_a$5 = threadItemCounts[index$1]) !== null && _a$5 !== void 0 ? _a$5 : 0) + 1;
			}
		}
	}
};
function isLiUnderOl(element) {
	return isElementOfType(element, "li") && isNodeOfType(element.parentNode, "ELEMENT_NODE") && isElementOfType(element.parentNode, "ol");
}
var listLevelThreadFormatHandler = {
	parse: function(format, element, context) {
		if (isElementOfType(element, "ol")) {
			var listFormat = context.listFormat;
			var threadItemCounts = listFormat.threadItemCounts;
			var depth = listFormat.levels.length;
			if (element.start == 1 || typeof threadItemCounts[depth] === "number" && element.start != threadItemCounts[depth] + 1) format.startNumberOverride = element.start;
			threadItemCounts[depth] = element.start - 1;
		}
	},
	apply: function(format, element, context) {
		var _a$5 = context.listFormat, threadItemCounts = _a$5.threadItemCounts;
		var depth = _a$5.nodeStack.length - 2;
		if (depth >= 0 && isElementOfType(element, "ol")) {
			var startNumber = format.startNumberOverride;
			if (typeof startNumber === "number") threadItemCounts[depth] = startNumber - 1;
			else if (typeof threadItemCounts[depth] != "number") threadItemCounts[depth] = 0;
			threadItemCounts.splice(depth + 1);
			element.start = threadItemCounts[depth] + 1;
		}
	}
};
var listStyleFormatHandler = {
	parse: function(format, element) {
		var listStylePosition = element.style.listStylePosition;
		var listStyleType = element.style.listStyleType;
		if (listStylePosition) format.listStylePosition = listStylePosition;
		if (listStyleType) format.listStyleType = listStyleType;
	},
	apply: function(format, element) {
		if (format.listStylePosition) element.style.listStylePosition = format.listStylePosition;
		if (format.listStyleType) element.style.listStyleType = format.listStyleType;
	}
};
var MarginKeys = [
	"marginTop",
	"marginRight",
	"marginBottom",
	"marginLeft"
];
var DefaultMarginKey = {
	ltr: {
		marginRight: "marginInlineEnd",
		marginLeft: "marginInlineStart"
	},
	rtl: {
		marginRight: "marginInlineStart",
		marginLeft: "marginInlineEnd"
	}
};
var LTR = {
	marginLeft: "marginRight",
	marginRight: "marginLeft",
	marginTop: "marginTop",
	marginBottom: "marginBottom"
};
var marginFormatHandler = {
	parse: function(format, element, _, defaultStyle) {
		MarginKeys.forEach(function(key) {
			var _a$5, _b$1;
			var alternativeKey = DefaultMarginKey[(_a$5 = format.direction) !== null && _a$5 !== void 0 ? _a$5 : "ltr"][key];
			var value = element.style[key] || defaultStyle[key] || (alternativeKey ? (_b$1 = defaultStyle[alternativeKey]) === null || _b$1 === void 0 ? void 0 : _b$1.toString() : "");
			if (value) switch (key) {
				case "marginTop":
				case "marginBottom":
					format[key] = value;
					break;
				case "marginLeft":
				case "marginRight":
					format[key] = format[key] ? parseValueWithUnit(format[key] || "", element) + parseValueWithUnit(value, element) + "px" : value;
					break;
			}
		});
	},
	apply: function(format, element, context) {
		MarginKeys.forEach(function(key) {
			var value = format[key];
			var ltrKey = format.direction == "rtl" ? LTR[key] : key;
			if (value != context.implicitFormat[ltrKey]) element.style[key] = value || "0";
		});
	}
};
var PaddingKeys = [
	"paddingTop",
	"paddingRight",
	"paddingBottom",
	"paddingLeft"
];
var AlternativeKeyLtr = { paddingLeft: "paddingInlineStart" };
var AlternativeKeyRtl = { paddingRight: "paddingInlineStart" };
var paddingFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		directionFormatHandler.parse(format, element, context, defaultStyle);
		PaddingKeys.forEach(function(key) {
			var _a$5, _b$1;
			var value = element.style[key];
			var alterativeKey = (format.direction == "rtl" ? AlternativeKeyRtl : AlternativeKeyLtr)[key];
			var defaultValue = ((_b$1 = (_a$5 = defaultStyle[key]) !== null && _a$5 !== void 0 ? _a$5 : alterativeKey ? defaultStyle[alterativeKey] : void 0) !== null && _b$1 !== void 0 ? _b$1 : "0px") + "";
			if (!value) value = defaultValue;
			if (!value || value == "0") value = "0px";
			if (value && value != defaultValue) format[key] = value;
		});
	},
	apply: function(format, element, context) {
		PaddingKeys.forEach(function(key) {
			var value = format[key];
			var defaultValue = void 0;
			if (element.tagName == "OL" || element.tagName == "UL") {
				if (format.direction == "rtl" && key == "paddingRight" || format.direction != "rtl" && key == "paddingLeft") defaultValue = "40px";
			}
			if (value && value != defaultValue) element.style[key] = value;
		});
	}
};
var roleFormatHandler = {
	parse: function(format, element) {
		var role = element.getAttribute("role");
		if (role) format.role = role;
	},
	apply: function(format, element) {
		if (format.role) element.setAttribute("role", format.role);
	}
};
var PercentageRegex = /[\d\.]+%/;
var sizeFormatHandler = {
	parse: function(format, element, context) {
		var width = element.style.width || tryParseSize(element, "width");
		var height = element.style.height || tryParseSize(element, "height");
		var maxWidth = element.style.maxWidth;
		var maxHeight = element.style.maxHeight;
		var minWidth = element.style.minWidth;
		var minHeight = element.style.minHeight;
		if (width) format.width = width;
		if (height) format.height = height;
		if (maxWidth) format.maxWidth = maxWidth;
		if (maxHeight) format.maxHeight = maxHeight;
		if (minWidth) format.minWidth = minWidth;
		if (minHeight) format.minHeight = minHeight;
	},
	apply: function(format, element) {
		if (format.width) element.style.width = format.width;
		if (format.height) element.style.height = format.height;
		if (format.maxWidth) element.style.maxWidth = format.maxWidth;
		if (format.maxHeight) element.style.maxHeight = format.maxHeight;
		if (format.minWidth) element.style.minWidth = format.minWidth;
		if (format.minHeight) element.style.minHeight = format.minHeight;
	}
};
function tryParseSize(element, attrName) {
	var attrValue = element.getAttribute(attrName);
	var value = parseInt(attrValue || "");
	return attrValue && PercentageRegex.test(attrValue) ? attrValue : Number.isNaN(value) || value == 0 ? void 0 : value + "px";
}
var strikeFormatHandler = {
	parse: function(format, element, context, defaultStyle) {
		var textDecoration = element.style.textDecoration || defaultStyle.textDecoration;
		if ((textDecoration === null || textDecoration === void 0 ? void 0 : textDecoration.indexOf("line-through")) >= 0) format.strikethrough = true;
	},
	apply: function(format, element) {
		if (format.strikethrough) wrapAllChildNodes(element, "s");
	}
};
var tableLayoutFormatHandler = {
	parse: function(format, element) {
		var tableLayout = element.style.tableLayout;
		if (tableLayout && tableLayout != "inherit") format.tableLayout = tableLayout;
	},
	apply: function(format, element) {
		if (format.tableLayout) element.style.tableLayout = format.tableLayout;
	}
};
var BorderCollapsed = "collapse";
var BorderSeparate = "separate";
var CellPadding = "cellPadding";
var defaultFormatHandlerMap = {
	aria: ariaFormatHandler,
	backgroundColor: backgroundColorFormatHandler,
	bold: boldFormatHandler,
	border: borderFormatHandler,
	borderBox: borderBoxFormatHandler,
	boxShadow: boxShadowFormatHandler,
	dataset: datasetFormatHandler,
	direction: directionFormatHandler,
	display: displayFormatHandler,
	float: floatFormatHandler,
	fontFamily: fontFamilyFormatHandler,
	fontSize: fontSizeFormatHandler,
	entity: entityFormatHandler,
	htmlAlign: htmlAlignFormatHandler,
	id: idFormatHandler,
	imageState: imageStateFormatHandler,
	italic: italicFormatHandler,
	letterSpacing: letterSpacingFormatHandler,
	lineHeight: lineHeightFormatHandler,
	link: linkFormatHandler,
	listItemThread: listItemThreadFormatHandler,
	listLevelThread: listLevelThreadFormatHandler,
	listStyle: listStyleFormatHandler,
	margin: marginFormatHandler,
	padding: paddingFormatHandler,
	role: roleFormatHandler,
	size: sizeFormatHandler,
	strike: strikeFormatHandler,
	superOrSubScript: superOrSubScriptFormatHandler,
	tableLayout: tableLayoutFormatHandler,
	tableSpacing: {
		parse: function(format, element) {
			if (element.style.borderCollapse == BorderCollapsed) format.borderCollapse = true;
			else if (element.getAttribute(CellPadding)) format.borderCollapse = true;
			if (element.style.borderCollapse == BorderSeparate) format.borderSeparate = true;
		},
		apply: function(format, element) {
			if (format.borderCollapse) {
				element.style.borderCollapse = BorderCollapsed;
				element.style.borderSpacing = "0";
				element.style.boxSizing = "border-box";
			} else if (format.borderSeparate) {
				element.style.borderCollapse = BorderSeparate;
				element.style.borderSpacing = "0";
				element.style.boxSizing = "border-box";
			}
		}
	},
	textAlign: {
		parse: function(format, element, context, defaultStyle) {
			var _a$5;
			directionFormatHandler.parse(format, element, context, defaultStyle);
			var textAlign = element.style.textAlign || defaultStyle.textAlign;
			if (element.tagName == "LI" && ((_a$5 = element.parentElement) === null || _a$5 === void 0 ? void 0 : _a$5.style.display) === "flex" && element.parentElement.style.flexDirection === "column" && element.style.alignSelf) textAlign = element.style.alignSelf;
			if (textAlign) format.textAlign = calcAlign(textAlign, format.direction);
		},
		apply: function(format, element) {
			var dir = format.direction == "rtl" ? "rtl" : "ltr";
			if (format.textAlign) {
				var parent_1 = element.parentElement;
				var parentTag = parent_1 === null || parent_1 === void 0 ? void 0 : parent_1.tagName;
				if (element.tagName == "LI" && parent_1 && (parentTag == "OL" || parentTag == "UL")) {
					element.style.alignSelf = format.textAlign;
					element.parentElement.style.flexDirection = "column";
					element.parentElement.style.display = "flex";
				} else element.style.textAlign = ResultMap$1[format.textAlign][dir];
			}
		}
	},
	textColor: {
		parse: function(format, element, context, defaultStyle) {
			var textColor = getColor(element, false, !!context.isDarkMode, context.darkColorHandler) || defaultStyle.color;
			if (textColor && textColor != "inherit") format.textColor = textColor;
		},
		apply: function(format, element, context) {
			var implicitColor = context.implicitFormat.textColor;
			if (format.textColor && format.textColor != implicitColor) setColor(element, format.textColor, false, !!context.isDarkMode, context.darkColorHandler);
		}
	},
	textColorOnTableCell: {
		parse: function(format, element) {
			if (element.style.color) delete format.textColor;
		},
		apply: function() {}
	},
	textIndent: {
		parse: function(format, element) {
			var textIndent = element.style.textIndent;
			if (textIndent) format.textIndent = textIndent;
		},
		apply: function(format, element) {
			if (format.textIndent) element.style.textIndent = format.textIndent;
		}
	},
	undeletableLink: {
		parse: function(format, element) {
			if (isElementOfType(element, "a") && isLinkUndeletable(element)) format.undeletable = true;
		},
		apply: function(format, element) {
			if (format.undeletable && isElementOfType(element, "a")) setLinkUndeletable(element, true);
		}
	},
	underline: {
		parse: function(format, element, context, defaultStyle) {
			var textDecoration = element.style.textDecoration || defaultStyle.textDecoration;
			if ((textDecoration === null || textDecoration === void 0 ? void 0 : textDecoration.indexOf("underline")) >= 0) format.underline = true;
			else if (element.tagName == "A" && textDecoration == "none") format.underline = false;
		},
		apply: function(format, element, context) {
			if (typeof format.underline === "undefined") return;
			if (!!context.implicitFormat.underline != !!format.underline) if (format.underline) wrapAllChildNodes(element, "u");
			else element.style.textDecoration = "none";
		}
	},
	verticalAlign: {
		parse: function(format, element) {
			switch (element.style.verticalAlign || element.getAttribute("valign")) {
				case "baseline":
				case "initial":
				case "super":
				case "sub":
				case "text-top":
				case "text-bottom":
				case "top":
					format.verticalAlign = "top";
					break;
				case "bottom":
					format.verticalAlign = "bottom";
					break;
				case "middle":
					format.verticalAlign = "middle";
					break;
			}
		},
		apply: function(format, element) {
			if (format.verticalAlign) element.style.verticalAlign = format.verticalAlign;
		}
	},
	whiteSpace: {
		parse: function(format, element, _, defaultStyle) {
			var whiteSpace = element.style.whiteSpace || defaultStyle.whiteSpace;
			if (shouldSetValue(whiteSpace, "normal", format.whiteSpace, defaultStyle.whiteSpace)) format.whiteSpace = whiteSpace;
		},
		apply: function(format, element, context) {
			var whiteSpace = context.implicitFormat.whiteSpace;
			if (format.whiteSpace != whiteSpace) element.style.whiteSpace = format.whiteSpace || "normal";
		}
	},
	wordBreak: {
		parse: function(format, element, _, defaultStyle) {
			var wordBreak = element.style.wordBreak || defaultStyle.wordBreak;
			if (wordBreak) format.wordBreak = wordBreak;
		},
		apply: function(format, element) {
			if (format.wordBreak) element.style.wordBreak = format.wordBreak;
		}
	}
};
var styleBasedSegmentFormats = [
	"letterSpacing",
	"fontFamily",
	"fontSize"
];
var elementBasedSegmentFormats = [
	"strike",
	"underline",
	"superOrSubScript",
	"italic",
	"bold"
];
var sharedBlockFormats = [
	"direction",
	"textAlign",
	"textIndent",
	"lineHeight",
	"whiteSpace"
];
var sharedContainerFormats = [
	"backgroundColor",
	"margin",
	"padding",
	"border"
];
var defaultFormatKeysPerCategory = {
	block: sharedBlockFormats,
	listItemThread: ["listItemThread"],
	listLevelThread: ["listLevelThread"],
	listItemElement: __spreadArray(__spreadArray([], __read(sharedBlockFormats), false), [
		"direction",
		"textAlign",
		"lineHeight",
		"margin",
		"listStyle"
	], false),
	listLevel: [
		"direction",
		"textAlign",
		"margin",
		"padding",
		"listStyle",
		"backgroundColor"
	],
	styleBasedSegment: __spreadArray(__spreadArray([], __read(styleBasedSegmentFormats), false), [
		"textColor",
		"backgroundColor",
		"lineHeight"
	], false),
	elementBasedSegment: elementBasedSegmentFormats,
	segment: __spreadArray(__spreadArray(__spreadArray([], __read(styleBasedSegmentFormats), false), __read(elementBasedSegmentFormats), false), [
		"textColor",
		"backgroundColor",
		"lineHeight"
	], false),
	segmentOnBlock: __spreadArray(__spreadArray(__spreadArray([], __read(styleBasedSegmentFormats), false), __read(elementBasedSegmentFormats), false), ["textColor"], false),
	segmentOnTableCell: __spreadArray(__spreadArray(__spreadArray([], __read(styleBasedSegmentFormats), false), __read(elementBasedSegmentFormats), false), ["textColorOnTableCell"], false),
	tableCell: [
		"border",
		"backgroundColor",
		"padding",
		"verticalAlign",
		"wordBreak",
		"textColor",
		"htmlAlign",
		"size"
	],
	tableRow: ["backgroundColor"],
	tableColumn: ["size"],
	table: [
		"aria",
		"id",
		"border",
		"backgroundColor",
		"display",
		"htmlAlign",
		"margin",
		"size",
		"tableLayout",
		"textColor",
		"direction",
		"role"
	],
	tableBorder: ["borderBox", "tableSpacing"],
	tableCellBorder: ["borderBox"],
	image: [
		"id",
		"size",
		"margin",
		"padding",
		"borderBox",
		"border",
		"boxShadow",
		"display",
		"float",
		"verticalAlign",
		"imageState"
	],
	link: [
		"link",
		"textColor",
		"underline",
		"display",
		"margin",
		"padding",
		"backgroundColor",
		"border",
		"size",
		"textAlign",
		"undeletableLink"
	],
	segmentUnderLink: ["textColor"],
	code: ["fontFamily", "display"],
	dataset: ["dataset"],
	divider: __spreadArray(__spreadArray(__spreadArray([], __read(sharedBlockFormats), false), __read(sharedContainerFormats), false), [
		"display",
		"size",
		"htmlAlign"
	], false),
	container: __spreadArray(__spreadArray([], __read(sharedContainerFormats), false), [
		"htmlAlign",
		"size",
		"display",
		"id"
	], false),
	entity: ["entity"],
	general: ["textColor", "backgroundColor"]
};
var defaultFormatParsers = getObjectKeys(defaultFormatHandlerMap).reduce(function(result, key) {
	result[key] = defaultFormatHandlerMap[key].parse;
	return result;
}, {});
var defaultFormatAppliers = getObjectKeys(defaultFormatHandlerMap).reduce(function(result, key) {
	result[key] = defaultFormatHandlerMap[key].apply;
	return result;
}, {});
function createModelToDomContext(editorContext) {
	var options = [];
	for (var _i = 1; _i < arguments.length; _i++) options[_i - 1] = arguments[_i];
	return createModelToDomContextWithConfig(createModelToDomConfig(options), editorContext);
}
function createModelToDomContextWithConfig(config, editorContext) {
	return Object.assign({}, editorContext, createModelToDomSelectionContext(), createModelToDomFormatContext(), createRewriteFromModelContext(), config);
}
function createModelToDomSelectionContext() {
	return { regularSelection: { current: {
		block: null,
		segment: null
	} } };
}
function createModelToDomFormatContext() {
	return {
		listFormat: {
			threadItemCounts: [],
			nodeStack: []
		},
		implicitFormat: {}
	};
}
function createRewriteFromModelContext() {
	return { rewriteFromModel: {
		addedBlockElements: [],
		removedBlockElements: []
	} };
}
function createModelToDomConfig(options) {
	return {
		modelHandlers: Object.assign.apply(Object, __spreadArray([{}, defaultContentModelHandlers], __read(options.map(function(x$1) {
			return x$1 === null || x$1 === void 0 ? void 0 : x$1.modelHandlerOverride;
		})), false)),
		formatAppliers: buildFormatAppliers(options.map(function(x$1) {
			return x$1 === null || x$1 === void 0 ? void 0 : x$1.formatApplierOverride;
		}), options.map(function(x$1) {
			return x$1 === null || x$1 === void 0 ? void 0 : x$1.additionalFormatAppliers;
		})),
		defaultModelHandlers: defaultContentModelHandlers,
		defaultFormatAppliers,
		metadataAppliers: Object.assign.apply(Object, __spreadArray([{}], __read(options.map(function(x$1) {
			return x$1 === null || x$1 === void 0 ? void 0 : x$1.metadataAppliers;
		})), false)),
		defaultContentModelFormatMap: Object.assign.apply(Object, __spreadArray([{}, defaultContentModelFormatMap], __read(options.map(function(x$1) {
			return x$1 === null || x$1 === void 0 ? void 0 : x$1.defaultContentModelFormatOverride;
		})), false))
	};
}
function buildFormatAppliers(overrides, additionalAppliersArray) {
	if (overrides === void 0) overrides = [];
	if (additionalAppliersArray === void 0) additionalAppliersArray = [];
	var combinedOverrides = Object.assign.apply(Object, __spreadArray([{}], __read(overrides), false));
	var result = getObjectKeys(defaultFormatKeysPerCategory).reduce(function(result$1, key) {
		var _a$5;
		result$1[key] = (_a$5 = defaultFormatKeysPerCategory[key].map(function(formatKey) {
			return combinedOverrides[formatKey] === void 0 ? defaultFormatAppliers[formatKey] : combinedOverrides[formatKey];
		})).concat.apply(_a$5, __spreadArray([], __read(additionalAppliersArray.map(function(appliers) {
			var _a$6;
			return (_a$6 = appliers === null || appliers === void 0 ? void 0 : appliers[key]) !== null && _a$6 !== void 0 ? _a$6 : [];
		})), false));
		return result$1;
	}, { text: [] });
	additionalAppliersArray.forEach(function(appliers) {
		if (appliers === null || appliers === void 0 ? void 0 : appliers.text) result.text = result.text.concat(appliers.text);
	});
	return result;
}
var brProcessor = function(group, element, context) {
	var _a$5;
	var br = createBr(context.segmentFormat);
	var _b$1 = __read(getRegularSelectionOffsets(context, element), 2), start = _b$1[0], end = _b$1[1];
	if (start >= 0) context.isInSelection = true;
	if (context.isInSelection) br.isSelected = true;
	var paragraph = addSegment(group, br, context.blockFormat);
	if (end >= 0) context.isInSelection = false;
	(_a$5 = context.domIndexer) === null || _a$5 === void 0 || _a$5.onSegment(element, paragraph, [br]);
};
var ContextStyles = [
	"marginLeft",
	"marginRight",
	"paddingLeft",
	"paddingRight"
];
var formatContainerProcessor = function(group, element, context) {
	stackFormat$1(context, {
		segment: "shallowCloneForBlock",
		paragraph: "shallowClone"
	}, function() {
		parseFormat(element, context.formatParsers.block, context.blockFormat, context);
		parseFormat(element, context.formatParsers.segmentOnBlock, context.segmentFormat, context);
		var format = __assign({}, context.blockFormat);
		parseFormat(element, context.formatParsers.container, format, context);
		var formatContainer = createFormatContainer(getDefaultStyle(element).display == "block" ? element.tagName.toLowerCase() : "div", format);
		ContextStyles.forEach(function(style) {
			delete context.blockFormat[style];
		});
		context.elementProcessors.child(formatContainer, element, context);
		if (element.style.fontSize && parseInt(element.style.fontSize) == 0) formatContainer.zeroFontSize = true;
		if (shouldFallbackToParagraph(formatContainer)) {
			var paragraph = formatContainer.blocks[0];
			if (formatContainer.zeroFontSize) paragraph.segmentFormat = Object.assign({}, paragraph.segmentFormat, { fontSize: "0" });
			Object.assign(paragraph.format, formatContainer.format);
			setParagraphNotImplicit(paragraph);
			addBlock(group, paragraph);
		} else addBlock(group, formatContainer);
	});
	addBlock(group, createParagraph(true, context.blockFormat));
};
function shouldFallbackToParagraph(formatContainer) {
	var firstChild = formatContainer.blocks[0];
	return formatContainer.tagName == "div" && formatContainer.blocks.length == 1 && firstChild.blockType == "Paragraph" && firstChild.isImplicit;
}
var SegmentDecoratorTags$1 = ["A", "CODE"];
function blockProcessor(group, element, context, segmentFormat) {
	var _a$5;
	var decorator = context.blockDecorator.tagName ? context.blockDecorator : void 0;
	var isSegmentDecorator = SegmentDecoratorTags$1.indexOf(element.tagName) >= 0;
	parseFormat(element, context.formatParsers.block, context.blockFormat, context);
	var blockFormat = __assign({}, context.blockFormat);
	parseFormat(element, context.formatParsers.container, blockFormat, context);
	ContextStyles.forEach(function(style) {
		if (blockFormat[style]) context.blockFormat[style] = blockFormat[style];
	});
	if (!isSegmentDecorator) {
		var paragraph = createParagraph(false, blockFormat, segmentFormat, decorator);
		(_a$5 = context.paragraphMap) === null || _a$5 === void 0 || _a$5.assignMarkerToModel(element, paragraph);
		addBlock(group, paragraph);
	}
	context.elementProcessors.child(group, element, context);
}
var FormatContainerTriggerStyles = [
	"marginBottom",
	"marginTop",
	"paddingBottom",
	"paddingTop",
	"paddingLeft",
	"paddingRight",
	"borderTopWidth",
	"borderBottomWidth",
	"borderLeftWidth",
	"borderRightWidth",
	"width",
	"height",
	"maxWidth",
	"maxHeight",
	"minWidth",
	"minHeight"
];
var FormatContainerTriggerAttributes = ["id"];
var ByPassFormatContainerTags = [
	"H1",
	"H2",
	"H3",
	"H4",
	"H5",
	"H6",
	"P",
	"A"
];
var SegmentDecoratorTags = ["A", "CODE"];
var knownElementProcessor = function(group, element, context) {
	var isBlock$1 = isBlockElement(element);
	if ((isBlock$1 || element.style.display == "inline-block") && shouldUseFormatContainer(element, context)) formatContainerProcessor(group, element, context);
	else if (isBlockEntityContainer(element)) context.elementProcessors.child(group, element, context);
	else if (isBlock$1) {
		var decorator = context.blockDecorator.tagName ? context.blockDecorator : void 0;
		var isSegmentDecorator = SegmentDecoratorTags.indexOf(element.tagName) >= 0;
		stackFormat$1(context, {
			segment: "shallowCloneForBlock",
			paragraph: "shallowClone"
		}, function() {
			var segmentFormat = {};
			parseFormat(element, context.formatParsers.segmentOnBlock, segmentFormat, context);
			Object.assign(context.segmentFormat, segmentFormat);
			blockProcessor(group, element, context, segmentFormat);
		});
		if (isBlock$1 && !isSegmentDecorator) addBlock(group, createParagraph(true, context.blockFormat, void 0, decorator));
	} else stackFormat$1(context, {
		segment: "shallowClone",
		paragraph: "shallowClone",
		link: "cloneFormat"
	}, function() {
		parseFormat(element, context.formatParsers.segment, context.segmentFormat, context);
		if (context.link.format.href && element.tagName != "A") parseFormat(element, context.formatParsers.segmentUnderLink, context.link.format, context);
		context.elementProcessors.child(group, element, context);
	});
};
function shouldUseFormatContainer(element, context) {
	if (ByPassFormatContainerTags.indexOf(element.tagName) >= 0) return false;
	var style = element.style;
	var defaultStyle = getDefaultStyle(element);
	var bgcolor = style.getPropertyValue("background-color");
	if (bgcolor && bgcolor != "transparent") return true;
	if (FormatContainerTriggerStyles.some(function(key) {
		return parseInt(style[key] || defaultStyle[key] || "") > 0;
	}) || FormatContainerTriggerAttributes.some(function(attr) {
		return element.hasAttribute(attr);
	})) return true;
	if (style.marginLeft == "auto" || style.marginRight == "auto") return true;
	if (element.getAttribute("align")) return true;
	return false;
}
var codeProcessor = function(group, element, context) {
	stackFormat$1(context, { code: "codeDefault" }, function() {
		parseFormat(element, context.formatParsers.code, context.code.format, context);
		knownElementProcessor(group, element, context);
	});
};
var delimiterProcessor = function(group, node, context) {
	var _a$5, _b$1;
	var range = ((_a$5 = context.selection) === null || _a$5 === void 0 ? void 0 : _a$5.type) == "range" ? context.selection.range : null;
	if (range) {
		if (node.contains(range.startContainer)) {
			context.isInSelection = true;
			addSelectionMarker$1(group, context);
		}
		if (((_b$1 = context.selection) === null || _b$1 === void 0 ? void 0 : _b$1.type) == "range" && node.contains(range.endContainer)) {
			if (!context.selection.range.collapsed) addSelectionMarker$1(group, context);
			context.isInSelection = false;
		}
	}
};
var elementProcessor = function(group, element, context) {
	var tagName = element.tagName.toLowerCase();
	(tryGetProcessorForEntity(element, context) || tryGetProcessorForDelimiter(element, context) || context.elementProcessors[tagName] || tagName.indexOf(":") >= 0 && context.elementProcessors.child || context.elementProcessors["*"])(group, element, context);
};
function tryGetProcessorForEntity(element, context) {
	return isEntityElement(element) || element.contentEditable == "false" ? context.elementProcessors.entity : null;
}
function tryGetProcessorForDelimiter(element, context) {
	return isEntityDelimiter(element) ? context.elementProcessors.delimiter : null;
}
var FontSizes = [
	"10px",
	"13px",
	"16px",
	"18px",
	"24px",
	"32px",
	"48px"
];
function getFontSize(size) {
	var intSize = parseInt(size || "");
	if (Number.isNaN(intSize)) return;
	else if (intSize < 1) return FontSizes[0];
	else if (intSize > FontSizes.length) return FontSizes[FontSizes.length - 1];
	else return FontSizes[intSize - 1];
}
var fontProcessor = function(group, element, context) {
	stackFormat$1(context, { segment: isBlockElement(element) ? "shallowCloneForBlock" : "shallowClone" }, function() {
		var fontFamily = element.getAttribute("face");
		var fontSize = getFontSize(element.getAttribute("size"));
		var textColor = element.getAttribute("color");
		var format = context.segmentFormat;
		if (fontFamily) format.fontFamily = fontFamily;
		if (fontSize) format.fontSize = fontSize;
		if (textColor) format.textColor = textColor;
		parseFormat(element, context.formatParsers.segment, context.segmentFormat, context);
		context.elementProcessors.child(group, element, context);
	});
};
var generalBlockProcessor = function(group, element, context) {
	var block = createGeneralBlock(element);
	var isSelectedBefore = context.isInSelection;
	stackFormat$1(context, {
		segment: "empty",
		paragraph: "empty",
		link: "empty"
	}, function() {
		addBlock(group, block);
		parseFormat(element, context.formatParsers.general, block.format, context);
		context.elementProcessors.child(block, element, context);
	});
	if (isSelectedBefore && context.isInSelection) block.isSelected = true;
};
var generalSegmentProcessor = function(group, element, context) {
	var _a$5;
	var segment = createGeneralSegment(element, context.segmentFormat);
	var isSelectedBefore = context.isInSelection;
	addDecorators(segment, context);
	var paragraph = addSegment(group, segment);
	(_a$5 = context.domIndexer) === null || _a$5 === void 0 || _a$5.onSegment(element, paragraph, [segment]);
	stackFormat$1(context, { segment: "empty" }, function() {
		parseFormat(element, context.formatParsers.general, segment.format, context);
		context.elementProcessors.child(segment, element, context);
	});
	if (isSelectedBefore && context.isInSelection) segment.isSelected = true;
};
var generalProcessor = function(group, element, context) {
	(isBlockElement(element) ? generalBlockProcessor : generalSegmentProcessor)(group, element, context);
};
var headingProcessor = function(group, element, context) {
	stackFormat$1(context, {
		segment: "shallowCloneForBlock",
		paragraph: "shallowClone",
		blockDecorator: "empty"
	}, function() {
		var segmentFormat = {};
		parseFormat(element, context.formatParsers.segmentOnBlock, segmentFormat, context);
		getObjectKeys(segmentFormat).forEach(function(key) {
			delete context.segmentFormat[key];
		});
		context.blockDecorator = createParagraphDecorator(element.tagName, segmentFormat);
		blockProcessor(group, element, context);
	});
	addBlock(group, createParagraph(true, context.blockFormat));
};
var hrProcessor = function(group, element, context) {
	stackFormat$1(context, { paragraph: "shallowClone" }, function() {
		parseFormat(element, context.formatParsers.divider, context.blockFormat, context);
		var hr = createDivider("hr", context.blockFormat);
		if (element.size) hr.size = element.size;
		if (context.isInSelection) hr.isSelected = true;
		addBlock(group, hr);
	});
};
var imageProcessor = function(group, element, context) {
	stackFormat$1(context, { segment: "shallowClone" }, function() {
		var _a$5, _b$1, _c$1;
		var imageFormat = context.segmentFormat;
		var src = (_a$5 = element.getAttribute("src")) !== null && _a$5 !== void 0 ? _a$5 : "";
		parseFormat(element, context.formatParsers.segment, imageFormat, context);
		parseFormat(element, context.formatParsers.image, imageFormat, context);
		parseFormat(element, context.formatParsers.block, context.blockFormat, context);
		var image = createImage(src, imageFormat);
		var alt = element.alt;
		var title = element.title;
		parseFormat(element, context.formatParsers.dataset, image.dataset, context);
		addDecorators(image, context);
		if (alt) image.alt = alt;
		if (title) image.title = title;
		if (context.isInSelection) image.isSelected = true;
		if (((_b$1 = context.selection) === null || _b$1 === void 0 ? void 0 : _b$1.type) == "image" && context.selection.image == element) {
			image.isSelectedAsImageSelection = true;
			image.isSelected = true;
		}
		var paragraph = addSegment(group, image);
		(_c$1 = context.domIndexer) === null || _c$1 === void 0 || _c$1.onSegment(element, paragraph, [image]);
	});
};
var linkProcessor = function(group, element, context) {
	var name = element.getAttribute("name");
	var href = element.getAttribute("href");
	if (name || href) {
		var isAnchor_1 = !!name && !href;
		stackFormat$1(context, { link: isAnchor_1 ? "empty" : "linkDefault" }, function() {
			parseFormat(element, context.formatParsers.link, context.link.format, context);
			parseFormat(element, context.formatParsers.dataset, context.link.dataset, context);
			if (isAnchor_1 && !element.firstChild) {
				var emptyText = createText("", context.segmentFormat, {
					dataset: context.link.dataset,
					format: context.link.format
				});
				if (context.isInSelection) emptyText.isSelected = true;
				addSegment(group, emptyText);
			} else knownElementProcessor(group, element, context);
		});
	} else knownElementProcessor(group, element, context);
};
var listItemProcessor = function(group, element, context) {
	var _a$5;
	var listFormat = context.listFormat;
	if (listFormat.listParent && listFormat.levels.length > 0) stackFormat$1(context, { segment: "shallowCloneForBlock" }, function() {
		parseFormat(element, context.formatParsers.segmentOnBlock, context.segmentFormat, context);
		var listItem = createListItem(listFormat.levels, context.segmentFormat);
		parseFormat(element, context.formatParsers.listItemElement, listItem.format, context);
		listFormat.listParent.blocks.push(listItem);
		parseFormat(element, context.formatParsers.listItemThread, listItem.levels[listItem.levels.length - 1].format, context);
		context.elementProcessors.child(listItem, element, context);
		var firstChild = listItem.blocks[0];
		if (listItem.blocks.length == 1 && firstChild.blockType == "Paragraph" && firstChild.isImplicit) {
			Object.assign(listItem.format, firstChild.format);
			firstChild.format = {};
		}
	});
	else {
		var currentBlocks = (_a$5 = listFormat.listParent) === null || _a$5 === void 0 ? void 0 : _a$5.blocks;
		var lastItem = currentBlocks === null || currentBlocks === void 0 ? void 0 : currentBlocks[(currentBlocks === null || currentBlocks === void 0 ? void 0 : currentBlocks.length) - 1];
		context.elementProcessors["*"]((lastItem === null || lastItem === void 0 ? void 0 : lastItem.blockType) == "BlockGroup" ? lastItem : group, element, context);
	}
};
var listProcessor = function(group, element, context) {
	stackFormat$1(context, {
		segment: "shallowCloneForBlock",
		paragraph: "shallowCloneForGroup"
	}, function() {
		var level = createListLevel(element.tagName, context.blockFormat);
		var listFormat = context.listFormat;
		parseFormat(element, context.formatParsers.dataset, level.dataset, context);
		parseFormat(element, context.formatParsers.listLevelThread, level.format, context);
		parseFormat(element, context.formatParsers.listLevel, level.format, context);
		parseFormat(element, context.formatParsers.segment, context.segmentFormat, context);
		var originalListParent = listFormat.listParent;
		listFormat.listParent = listFormat.listParent || group;
		listFormat.levels.push(level);
		try {
			context.elementProcessors.child(group, element, context);
		} finally {
			listFormat.levels.pop();
			listFormat.listParent = originalListParent;
		}
	});
};
var pProcessor = function(group, element, context) {
	stackFormat$1(context, {
		blockDecorator: "empty",
		segment: "shallowCloneForBlock",
		paragraph: "shallowClone"
	}, function() {
		context.blockDecorator = createParagraphDecorator(element.tagName);
		var segmentFormat = {};
		parseFormat(element, context.formatParsers.segmentOnBlock, segmentFormat, context);
		Object.assign(context.segmentFormat, segmentFormat);
		blockProcessor(group, element, context, segmentFormat);
	});
	addBlock(group, createParagraph(true, context.blockFormat));
};
var textProcessor = function(group, textNode, context) {
	if (context.formatParsers.text.length > 0) stackFormat$1(context, { segment: "shallowClone" }, function() {
		context.formatParsers.text.forEach(function(parser) {
			parser(context.segmentFormat, textNode, context);
			internalTextProcessor(group, textNode, context);
		});
	});
	else internalTextProcessor(group, textNode, context);
};
function internalTextProcessor(group, textNode, context) {
	var _a$5;
	var paragraph = ensureParagraph(group, context.blockFormat);
	var segmentCount = paragraph.segments.length;
	context.elementProcessors.textWithSelection(group, textNode, context);
	var newSegments = paragraph.segments.slice(segmentCount);
	(_a$5 = context.domIndexer) === null || _a$5 === void 0 || _a$5.onSegment(textNode, paragraph, newSegments.filter(function(x$1) {
		return (x$1 === null || x$1 === void 0 ? void 0 : x$1.segmentType) == "Text";
	}));
}
var textWithSelectionProcessor = function(group, textNode, context) {
	var txt = textNode.nodeValue || "";
	var offsets = getRegularSelectionOffsets(context, textNode);
	var txtStartOffset = offsets[0];
	var txtEndOffset = offsets[1];
	if (txtStartOffset >= 0) {
		var subText = txt.substring(0, txtStartOffset);
		addTextSegment(group, subText, context);
		context.isInSelection = true;
		addSelectionMarker$1(group, context, textNode, txtStartOffset);
		txt = txt.substring(txtStartOffset);
		txtEndOffset -= txtStartOffset;
	}
	if (txtEndOffset >= 0) {
		var subText = txt.substring(0, txtEndOffset);
		addTextSegment(group, subText, context);
		if (context.selection && (context.selection.type != "range" || !context.selection.range.collapsed)) addSelectionMarker$1(group, context, textNode, offsets[1]);
		context.isInSelection = false;
		txt = txt.substring(txtEndOffset);
	}
	addTextSegment(group, txt, context);
};
var defaultProcessorMap = {
	a: linkProcessor,
	b: knownElementProcessor,
	blockquote: knownElementProcessor,
	br: brProcessor,
	code: codeProcessor,
	dd: formatContainerProcessor,
	del: knownElementProcessor,
	div: knownElementProcessor,
	dl: formatContainerProcessor,
	dt: formatContainerProcessor,
	em: knownElementProcessor,
	font: fontProcessor,
	i: knownElementProcessor,
	img: imageProcessor,
	h1: headingProcessor,
	h2: headingProcessor,
	h3: headingProcessor,
	h4: headingProcessor,
	h5: headingProcessor,
	h6: headingProcessor,
	hr: hrProcessor,
	li: listItemProcessor,
	ol: listProcessor,
	p: pProcessor,
	pre: formatContainerProcessor,
	s: knownElementProcessor,
	section: knownElementProcessor,
	span: knownElementProcessor,
	strike: knownElementProcessor,
	strong: knownElementProcessor,
	sub: knownElementProcessor,
	sup: knownElementProcessor,
	table: tableProcessor,
	u: knownElementProcessor,
	ul: listProcessor,
	"*": generalProcessor,
	"#text": textProcessor,
	textWithSelection: textWithSelectionProcessor,
	element: elementProcessor,
	entity: entityProcessor,
	child: childProcessor$1,
	delimiter: delimiterProcessor
};
function createDomToModelContext(editorContext) {
	var options = [];
	for (var _i = 1; _i < arguments.length; _i++) options[_i - 1] = arguments[_i];
	return createDomToModelContextWithConfig(createDomToModelConfig(options), editorContext);
}
function createDomToModelContextWithConfig(config, editorContext) {
	return Object.assign({}, editorContext, createDomToModelSelectionContext(), createDomToModelFormatContext(editorContext === null || editorContext === void 0 ? void 0 : editorContext.isRootRtl), createDomToModelDecoratorContext(), config);
}
function createDomToModelSelectionContext() {
	return { isInSelection: false };
}
function createDomToModelFormatContext(isRootRtl) {
	return {
		blockFormat: isRootRtl ? { direction: "rtl" } : {},
		segmentFormat: {},
		listFormat: {
			levels: [],
			threadItemCounts: []
		}
	};
}
function createDomToModelDecoratorContext() {
	return {
		link: {
			format: {},
			dataset: {}
		},
		code: { format: {} },
		blockDecorator: {
			format: {},
			tagName: ""
		}
	};
}
function createDomToModelConfig(options) {
	return {
		elementProcessors: Object.assign.apply(Object, __spreadArray([{}, defaultProcessorMap], __read(options.map(function(x$1) {
			return x$1 === null || x$1 === void 0 ? void 0 : x$1.processorOverride;
		})), false)),
		formatParsers: buildFormatParsers(options.map(function(x$1) {
			return x$1 === null || x$1 === void 0 ? void 0 : x$1.formatParserOverride;
		}), options.map(function(x$1) {
			return x$1 === null || x$1 === void 0 ? void 0 : x$1.additionalFormatParsers;
		})),
		defaultElementProcessors: defaultProcessorMap,
		defaultFormatParsers,
		processNonVisibleElements: options.some(function(x$1) {
			return !!(x$1 === null || x$1 === void 0 ? void 0 : x$1.processNonVisibleElements);
		})
	};
}
function buildFormatParsers(overrides, additionalParsersArray) {
	if (overrides === void 0) overrides = [];
	if (additionalParsersArray === void 0) additionalParsersArray = [];
	var combinedOverrides = Object.assign.apply(Object, __spreadArray([{}], __read(overrides), false));
	var result = getObjectKeys(defaultFormatKeysPerCategory).reduce(function(result$1, key) {
		var _a$5;
		result$1[key] = (_a$5 = defaultFormatKeysPerCategory[key].map(function(formatKey) {
			return combinedOverrides[formatKey] === void 0 ? defaultFormatParsers[formatKey] : combinedOverrides[formatKey];
		})).concat.apply(_a$5, __spreadArray([], __read(additionalParsersArray.map(function(parsers) {
			var _a$6;
			return (_a$6 = parsers === null || parsers === void 0 ? void 0 : parsers[key]) !== null && _a$6 !== void 0 ? _a$6 : [];
		})), false));
		return result$1;
	}, { text: [] });
	additionalParsersArray.forEach(function(parsers) {
		if (parsers === null || parsers === void 0 ? void 0 : parsers.text) result.text = result.text.concat(parsers.text);
	});
	return result;
}
function normalizeSegmentFormat(format, environment) {
	var _a$5, _b$1;
	var span = document.createElement("span");
	var segment = createText("text", format);
	var domToModelContext = createDomToModelContextWithConfig(environment.domToModelSettings.calculated);
	var modelToDomContext = createModelToDomContextWithConfig(environment.modelToDomSettings.calculated);
	var model = createContentModelDocument();
	modelToDomContext.modelHandlers.segment(span.ownerDocument, span, segment, modelToDomContext, []);
	domToModelContext.elementProcessors.element(model, span, domToModelContext);
	return (_b$1 = (_a$5 = ensureParagraph(model).segments[0]) === null || _a$5 === void 0 ? void 0 : _a$5.format) !== null && _b$1 !== void 0 ? _b$1 : format;
}
var NumberingListType = {
	Min: 1,
	Decimal: 1,
	DecimalDash: 2,
	DecimalParenthesis: 3,
	DecimalDoubleParenthesis: 4,
	LowerAlpha: 5,
	LowerAlphaParenthesis: 6,
	LowerAlphaDoubleParenthesis: 7,
	LowerAlphaDash: 8,
	UpperAlpha: 9,
	UpperAlphaParenthesis: 10,
	UpperAlphaDoubleParenthesis: 11,
	UpperAlphaDash: 12,
	LowerRoman: 13,
	LowerRomanParenthesis: 14,
	LowerRomanDoubleParenthesis: 15,
	LowerRomanDash: 16,
	UpperRoman: 17,
	UpperRomanParenthesis: 18,
	UpperRomanDoubleParenthesis: 19,
	UpperRomanDash: 20,
	Max: 20
};
var CharCodeOfA = 65;
var RomanValues = {
	M: 1e3,
	CM: 900,
	D: 500,
	CD: 400,
	C: 100,
	XC: 90,
	L: 50,
	XL: 40,
	X: 10,
	IX: 9,
	V: 5,
	IV: 4,
	I: 1
};
function getOrderedListNumberStr(styleType, listNumber) {
	switch (styleType) {
		case NumberingListType.LowerAlpha:
		case NumberingListType.LowerAlphaDash:
		case NumberingListType.LowerAlphaDoubleParenthesis:
		case NumberingListType.LowerAlphaParenthesis: return convertDecimalsToAlpha(listNumber, true);
		case NumberingListType.UpperAlpha:
		case NumberingListType.UpperAlphaDash:
		case NumberingListType.UpperAlphaDoubleParenthesis:
		case NumberingListType.UpperAlphaParenthesis: return convertDecimalsToAlpha(listNumber, false);
		case NumberingListType.LowerRoman:
		case NumberingListType.LowerRomanDash:
		case NumberingListType.LowerRomanDoubleParenthesis:
		case NumberingListType.LowerRomanParenthesis: return convertDecimalsToRoman(listNumber, true);
		case NumberingListType.UpperRoman:
		case NumberingListType.UpperRomanDash:
		case NumberingListType.UpperRomanDoubleParenthesis:
		case NumberingListType.UpperRomanParenthesis: return convertDecimalsToRoman(listNumber, false);
		default: return listNumber + "";
	}
}
function convertDecimalsToAlpha(decimal, isLowerCase) {
	var alpha = "";
	decimal--;
	while (decimal >= 0) {
		alpha = String.fromCharCode(decimal % 26 + CharCodeOfA) + alpha;
		decimal = Math.floor(decimal / 26) - 1;
	}
	return isLowerCase ? alpha.toLowerCase() : alpha;
}
function convertDecimalsToRoman(decimal, isLowerCase) {
	var e_1, _a$5;
	var romanValue = "";
	try {
		for (var _b$1 = __values(getObjectKeys(RomanValues)), _c$1 = _b$1.next(); !_c$1.done; _c$1 = _b$1.next()) {
			var i$1 = _c$1.value;
			var timesRomanCharAppear = Math.floor(decimal / RomanValues[i$1]);
			decimal = decimal - timesRomanCharAppear * RomanValues[i$1];
			romanValue = romanValue + i$1.repeat(timesRomanCharAppear);
		}
	} catch (e_1_1) {
		e_1 = { error: e_1_1 };
	} finally {
		try {
			if (_c$1 && !_c$1.done && (_a$5 = _b$1.return)) _a$5.call(_b$1);
		} finally {
			if (e_1) throw e_1.error;
		}
	}
	return isLowerCase ? romanValue.toLocaleLowerCase() : romanValue;
}
var BulletListType = {
	Min: 1,
	Disc: 1,
	Dash: 2,
	Square: 3,
	ShortArrow: 4,
	LongArrow: 5,
	UnfilledArrow: 6,
	Hyphen: 7,
	DoubleLongArrow: 8,
	Circle: 9,
	BoxShadow: 10,
	Xrhombus: 11,
	CheckMark: 12,
	Max: 12
};
var DefaultOrderedListStyles = [
	NumberingListType.Decimal,
	NumberingListType.LowerAlpha,
	NumberingListType.LowerRoman
];
var DefaultUnorderedListStyles = [
	BulletListType.Disc,
	BulletListType.Circle,
	BulletListType.Square
];
var OrderedListStyleRevertMap = {
	"lower-alpha": NumberingListType.LowerAlpha,
	"lower-latin": NumberingListType.LowerAlpha,
	"upper-alpha": NumberingListType.UpperAlpha,
	"upper-latin": NumberingListType.UpperAlpha,
	"lower-roman": NumberingListType.LowerRoman,
	"upper-roman": NumberingListType.UpperRoman
};
var UnorderedListStyleRevertMap = {
	disc: BulletListType.Disc,
	circle: BulletListType.Circle,
	square: BulletListType.Square
};
function getAutoListStyleType(listType, metadata, depth, existingStyleType) {
	var orderedStyleType = metadata.orderedStyleType, unorderedStyleType = metadata.unorderedStyleType, applyListStyleFromLevel = metadata.applyListStyleFromLevel;
	if (listType == "OL") return typeof orderedStyleType == "number" ? orderedStyleType : applyListStyleFromLevel ? DefaultOrderedListStyles[depth % DefaultOrderedListStyles.length] : existingStyleType ? OrderedListStyleRevertMap[existingStyleType] : void 0;
	else return typeof unorderedStyleType == "number" ? unorderedStyleType : applyListStyleFromLevel ? DefaultUnorderedListStyles[depth % DefaultUnorderedListStyles.length] : existingStyleType ? UnorderedListStyleRevertMap[existingStyleType] : void 0;
}
function isBold(boldStyle) {
	return !!boldStyle && (boldStyle == "bold" || boldStyle == "bolder" || parseInt(boldStyle) >= 600);
}
function getDOMInsertPointRect(doc, pos) {
	var _a$5, _b$1;
	var range = doc.createRange();
	return (_b$1 = (_a$5 = tryGetRectFromPos(pos, range)) !== null && _a$5 !== void 0 ? _a$5 : tryGetRectFromPos(pos = normalizeInsertPoint(pos), range)) !== null && _b$1 !== void 0 ? _b$1 : tryGetRectFromNode(pos.node);
}
function normalizeInsertPoint(pos) {
	var node = pos.node, offset = pos.offset;
	while (node.lastChild) if (offset == node.childNodes.length) {
		node = node.lastChild;
		offset = node.childNodes.length;
	} else {
		node = node.childNodes[offset];
		offset = 0;
	}
	return {
		node,
		offset
	};
}
function tryGetRectFromPos(pos, range) {
	var node = pos.node, offset = pos.offset;
	range.setStart(node, offset);
	range.setEnd(node, offset);
	var rect = normalizeRect(range.getBoundingClientRect());
	if (rect) return rect;
	else {
		var rects = range.getClientRects && range.getClientRects();
		return rects && rects.length == 1 ? normalizeRect(rects[0]) : null;
	}
}
function tryGetRectFromNode(node) {
	return isNodeOfType(node, "ELEMENT_NODE") && node.getBoundingClientRect ? normalizeRect(node.getBoundingClientRect()) : null;
}
var CTRL_CHAR_CODE = "Control";
var ALT_CHAR_CODE = "Alt";
var META_CHAR_CODE = "Meta";
var CursorMovingKeys = new Set([
	"ArrowUp",
	"ArrowDown",
	"ArrowLeft",
	"ArrowRight",
	"Home",
	"End",
	"PageUp",
	"PageDown"
]);
function isModifierKey(event$1) {
	var isCtrlKey = event$1.ctrlKey || event$1.key === CTRL_CHAR_CODE;
	var isAltKey = event$1.altKey || event$1.key === ALT_CHAR_CODE;
	var isMetaKey = event$1.metaKey || event$1.key === META_CHAR_CODE;
	return isCtrlKey || isAltKey || isMetaKey;
}
function isCharacterValue(event$1) {
	return !isModifierKey(event$1) && !!event$1.key && event$1.key.length == 1;
}
function isCursorMovingKey(event$1) {
	return CursorMovingKeys.has(event$1.key);
}
var BorderStyles = [
	"none",
	"hidden",
	"dotted",
	"dashed",
	"solid",
	"double",
	"groove",
	"ridge",
	"inset",
	"outset"
];
var BorderSizeRegex = /^(thin|medium|thick|[\d\.]+\w*)$/;
function extractBorderValues(combinedBorder) {
	var result = {};
	(combinedBorder || "").replace(/, /g, ",").split(" ").forEach(function(v$1) {
		if (BorderStyles.indexOf(v$1) >= 0 && !result.style) result.style = v$1;
		else if (BorderSizeRegex.test(v$1) && !result.width) result.width = v$1;
		else if (v$1 && !result.color) result.color = v$1;
	});
	return result;
}
function combineBorderValue(value) {
	return [
		value.width || "",
		value.style || "",
		value.color || ""
	].join(" ").trim() || "none";
}
var SPACES_REGEX = /[\u2000\u2009\u200a​\u200b​\u202f\u205f​\u3000\s\t\r\n]/gm;
var PUNCTUATIONS = ".,?!:\"()[]\\/";
function isPunctuation(char) {
	return PUNCTUATIONS.indexOf(char) >= 0;
}
function isSpace(char) {
	var _a$5;
	var code = (_a$5 = char === null || char === void 0 ? void 0 : char.charCodeAt(0)) !== null && _a$5 !== void 0 ? _a$5 : 0;
	return code == 160 || code == 32 || SPACES_REGEX.test(char);
}
function normalizeText(txt, isForward) {
	return txt.replace(isForward ? /^\u0020+/ : /\u0020+$/, "\xA0");
}
function parseTableCells(table) {
	var trs = toArray(table.rows);
	var cells = trs.map(function(row) {
		return [];
	});
	trs.forEach(function(tr, rowIndex) {
		for (var sourceCol = 0, targetCol = 0; sourceCol < tr.cells.length; sourceCol++) {
			for (; cells[rowIndex][targetCol] !== void 0; targetCol++);
			var td = tr.cells[sourceCol];
			for (var colSpan = 0; colSpan < td.colSpan; colSpan++, targetCol++) for (var rowSpan = 0; rowSpan < td.rowSpan; rowSpan++) if (cells[rowIndex + rowSpan]) cells[rowIndex + rowSpan][targetCol] = colSpan == 0 ? rowSpan == 0 ? td : "spanTop" : rowSpan == 0 ? "spanLeft" : "spanBoth";
		}
		for (var col = 0; col < cells[rowIndex].length; col++) cells[rowIndex][col] = cells[rowIndex][col] || null;
	});
	return cells;
}
function readFile(file, callback) {
	try {
		if (file) {
			var reader_1 = new FileReader();
			reader_1.onload = function() {
				callback(reader_1.result);
			};
			reader_1.onerror = function() {
				callback(null);
			};
			reader_1.readAsDataURL(file);
		}
	} catch (_a$5) {
		callback(null);
	}
}
function transformColor(rootNode, includeSelf, direction, darkColorHandler) {
	var toDarkMode = direction == "lightToDark";
	var transformer = function(element) {
		var textColor = getColor(element, false, !toDarkMode, darkColorHandler);
		var backColor = getColor(element, true, !toDarkMode, darkColorHandler);
		setColor(element, textColor, false, toDarkMode, darkColorHandler);
		setColor(element, backColor, true, toDarkMode, darkColorHandler);
	};
	iterateElements(rootNode, transformer, includeSelf);
}
function iterateElements(root$12, transformer, includeSelf) {
	if (includeSelf && isHTMLElement(root$12)) transformer(root$12);
	for (var child$1 = root$12.firstChild; child$1; child$1 = child$1.nextSibling) {
		if (isHTMLElement(child$1)) transformer(child$1);
		iterateElements(child$1, transformer);
	}
}
function isHTMLElement(node) {
	var htmlElement = node;
	return node.nodeType == Node.ELEMENT_NODE && !!htmlElement.style;
}
function normalizeFontFamily(fontFamily) {
	var existingQuotedFontsRegex = /(".*?")|('.*?')/g;
	var match = existingQuotedFontsRegex.exec(fontFamily);
	var start = 0;
	var result = [];
	while (match) {
		process(fontFamily, result, start, match.index);
		start = match.index + match[0].length;
		result.push(match[0]);
		match = existingQuotedFontsRegex.exec(fontFamily);
	}
	process(fontFamily, result, start, fontFamily.length);
	return result.join(", ");
}
function process(fontFamily, result, start, end) {
	fontFamily.substring(start, end).split(",").forEach(function(family) {
		family = family.trim();
		if (family) if (/[^a-zA-Z0-9\-]/.test(family)) result.push("\"" + family + "\"");
		else result.push(family);
	});
}
var _a$4;
var ContentHandlers = (_a$4 = {}, _a$4["text/html"] = function(data, value) {
	return data.rawHtml = value;
}, _a$4["text/plain"] = function(data, value) {
	return data.text = value;
}, _a$4["text/*"] = function(data, value, type) {
	return !!type && (data.customValues[type] = value);
}, _a$4["text/link-preview"] = tryParseLinkPreview, _a$4["text/uri-list"] = function(data, value) {
	return data.text = value;
}, _a$4);
function extractClipboardItems(items, allowedCustomPasteType) {
	var data = {
		types: [],
		text: "",
		image: null,
		files: [],
		rawHtml: null,
		customValues: {},
		pasteNativeEvent: true
	};
	return Promise.all((items || []).map(function(item) {
		var type = item.type;
		if (type.indexOf("image/") == 0 && !data.image && item.kind == "file") {
			data.types.push(type);
			data.image = item.getAsFile();
			return new Promise(function(resolve) {
				if (data.image) readFile(data.image, function(dataUrl) {
					data.imageDataUri = dataUrl;
					resolve();
				});
				else resolve();
			});
		} else if (item.kind == "file") return new Promise(function(resolve) {
			var file = item.getAsFile();
			if (!!file) {
				data.types.push(type);
				data.files.push(file);
			}
			resolve();
		});
		else {
			var customType_1 = getAllowedCustomType(type, allowedCustomPasteType);
			var handler_1 = ContentHandlers[type] || (customType_1 ? ContentHandlers["text/*"] : null);
			return new Promise(function(resolve) {
				return handler_1 ? item.getAsString(function(value) {
					data.types.push(type);
					handler_1(data, value, customType_1);
					resolve();
				}) : resolve();
			});
		}
	})).then(function() {
		return data;
	});
}
function tryParseLinkPreview(data, value) {
	try {
		data.customValues["link-preview"] = value;
		data.linkPreview = JSON.parse(value);
	} catch (_a$5) {}
}
function getAllowedCustomType(type, allowedCustomPasteType) {
	var textType = type.indexOf("text/") == 0 ? type.substring(5) : null;
	var index$1 = allowedCustomPasteType && textType ? allowedCustomPasteType.indexOf(textType) : -1;
	return textType && index$1 >= 0 ? textType : void 0;
}
function cacheGetEventData(event$1, key, getter) {
	var result = event$1.eventDataCache && event$1.eventDataCache.hasOwnProperty(key) ? event$1.eventDataCache[key] : getter(event$1);
	event$1.eventDataCache = event$1.eventDataCache || {};
	event$1.eventDataCache[key] = result;
	return result;
}
function getParagraphMarker(element) {
	return getHiddenProperty(element, "paragraphMarker");
}
function setParagraphMarker(element, marker) {
	setHiddenProperty(element, "paragraphMarker", marker);
}
var SplittingTags = [
	"BR",
	"HR",
	"IMG"
];
function getRangesByText(root$12, text, matchCase, wholeWord, editableOnly) {
	var context = {
		text: matchCase ? text : text.toLowerCase(),
		matchCase,
		wholeWord,
		result: [],
		paragraphText: "",
		indexes: [],
		editableOnly: !!editableOnly
	};
	if (context.text) iterateTextNodes(root$12, context);
	return context.result;
}
function isSplittingElement(element) {
	return isBlockElement(element) || SplittingTags.indexOf(element.tagName) >= 0;
}
function iterateTextNodes(root$12, context) {
	var canSearchText = !context.editableOnly || root$12.isContentEditable;
	for (var node = root$12.firstChild; node; node = node.nextSibling) if (isNodeOfType(node, "TEXT_NODE") && canSearchText) {
		var nodeText = context.matchCase ? node.nodeValue || "" : (node.nodeValue || "").toLowerCase();
		if (nodeText) {
			context.paragraphText += nodeText;
			context.indexes.push({
				node,
				length: nodeText.length
			});
		}
	} else if (isNodeOfType(node, "ELEMENT_NODE")) {
		if (context.paragraphText && isSplittingElement(node)) search(root$12.ownerDocument, context);
		iterateTextNodes(node, context);
	}
	if (context.paragraphText && isSplittingElement(root$12)) search(root$12.ownerDocument, context);
}
function search(doc, context) {
	var offset;
	var startIndex = 0;
	while ((offset = context.paragraphText.indexOf(context.text, startIndex)) > -1) {
		if (!context.wholeWord || (offset == 0 || isSpaceOrPunctuation(context.paragraphText[offset - 1])) && (offset + context.text.length == context.paragraphText.length || isSpaceOrPunctuation(context.paragraphText[offset + context.text.length]))) {
			var start = findNodeAndOffset(context.indexes, offset, false);
			var end = findNodeAndOffset(context.indexes, offset + context.text.length, true);
			if (start && end) {
				var range = doc.createRange();
				range.setStart(start.node, start.offset);
				range.setEnd(end.node, end.offset);
				context.result.push(range);
			}
		}
		startIndex = offset + context.text.length;
	}
	context.paragraphText = "";
	context.indexes = [];
}
function isSpaceOrPunctuation(char) {
	return isSpace(char) || isPunctuation(char);
}
function findNodeAndOffset(lengths, offset, isEnd) {
	var currentIndex = 0;
	for (var i$1 = 0; i$1 < lengths.length; i$1++) {
		var segmentLength = lengths[i$1].length;
		if (isEnd ? currentIndex + segmentLength >= offset : currentIndex + segmentLength > offset) return {
			node: lengths[i$1].node,
			offset: offset - currentIndex
		};
		currentIndex += segmentLength;
	}
	return null;
}
function isBlockGroupOfType(input, type) {
	var item = input;
	return (item === null || item === void 0 ? void 0 : item.blockGroupType) == type;
}
function iterateSelections(group, callback, option) {
	var internalCallback = (group.blockGroupType == "Document" ? group.persistCache : false) ? callback : function(path, tableContext, block, segments) {
		var _a$5;
		if (!!((_a$5 = block) === null || _a$5 === void 0 ? void 0 : _a$5.cachedElement)) delete block.cachedElement;
		return callback(path, tableContext, block, segments);
	};
	internalIterateSelections([group], internalCallback, option);
}
function internalIterateSelections(path, callback, option, table, treatAllAsSelect) {
	var parent = path[0];
	var includeListFormatHolder = (option === null || option === void 0 ? void 0 : option.includeListFormatHolder) || "allSegments";
	var contentUnderSelectedTableCell = (option === null || option === void 0 ? void 0 : option.contentUnderSelectedTableCell) || "include";
	var contentUnderSelectedGeneralElement = (option === null || option === void 0 ? void 0 : option.contentUnderSelectedGeneralElement) || "contentOnly";
	var hasSelectedSegment = false;
	var hasUnselectedSegment = false;
	for (var i$1 = 0; i$1 < parent.blocks.length; i$1++) {
		var block = parent.blocks[i$1];
		switch (block.blockType) {
			case "BlockGroup":
				var newPath = __spreadArray([block], __read(path), false);
				if (block.blockGroupType == "General") {
					var isSelected$1 = treatAllAsSelect || block.isSelected;
					var handleGeneralContent = !isSelected$1 || contentUnderSelectedGeneralElement == "both" || contentUnderSelectedGeneralElement == "contentOnly";
					var handleGeneralElement = isSelected$1 && (contentUnderSelectedGeneralElement == "both" || contentUnderSelectedGeneralElement == "generalElementOnly" || block.blocks.length == 0);
					if (handleGeneralContent && internalIterateSelections(newPath, callback, option, table, isSelected$1) || handleGeneralElement && callback(path, table, block)) return true;
				} else if (internalIterateSelections(newPath, callback, option, table, treatAllAsSelect)) return true;
				break;
			case "Table":
				var rows = block.rows;
				var isWholeTableSelected$1 = rows.every(function(row$1) {
					return row$1.cells.every(function(cell$1) {
						return cell$1.isSelected;
					});
				});
				if (contentUnderSelectedTableCell != "include" && isWholeTableSelected$1) {
					if (callback(path, table, block)) return true;
				} else for (var rowIndex = 0; rowIndex < rows.length; rowIndex++) {
					var row = rows[rowIndex];
					for (var colIndex = 0; colIndex < row.cells.length; colIndex++) {
						var cell = row.cells[colIndex];
						if (!cell) continue;
						var newTable = {
							table: block,
							rowIndex,
							colIndex,
							isWholeTableSelected: isWholeTableSelected$1
						};
						if (cell.isSelected && callback(path, newTable)) return true;
						if (!cell.isSelected || contentUnderSelectedTableCell != "ignoreForTableOrCell") {
							var newPath_1 = __spreadArray([cell], __read(path), false);
							var isSelected$1 = treatAllAsSelect || cell.isSelected;
							if (internalIterateSelections(newPath_1, callback, option, newTable, isSelected$1)) return true;
						}
					}
				}
				break;
			case "Paragraph":
				var segments = [];
				for (var i_1 = 0; i_1 < block.segments.length; i_1++) {
					var segment = block.segments[i_1];
					var isSelected$1 = treatAllAsSelect || segment.isSelected;
					if (segment.segmentType == "General") {
						var handleGeneralContent = !isSelected$1 || contentUnderSelectedGeneralElement == "both" || contentUnderSelectedGeneralElement == "contentOnly";
						var handleGeneralElement = isSelected$1 && (contentUnderSelectedGeneralElement == "both" || contentUnderSelectedGeneralElement == "generalElementOnly" || segment.blocks.length == 0);
						if (handleGeneralContent && internalIterateSelections(__spreadArray([segment], __read(path), false), callback, option, table, isSelected$1)) return true;
						if (handleGeneralElement) segments.push(segment);
					} else if (isSelected$1) segments.push(segment);
					if (isSelected$1) hasSelectedSegment = true;
					else hasUnselectedSegment = true;
				}
				if (segments.length > 0 && callback(path, table, block, segments)) return true;
				break;
			case "Divider":
			case "Entity":
				if ((treatAllAsSelect || block.isSelected) && callback(path, table, block)) return true;
				break;
		}
	}
	if (includeListFormatHolder != "never" && parent.blockGroupType == "ListItem" && hasSelectedSegment && (!hasUnselectedSegment || includeListFormatHolder == "anySegment") && callback(path, table, void 0, [parent.formatHolder])) return true;
	return false;
}
function getClosestAncestorBlockGroupIndex(path, blockGroupTypes, stopTypes, isValidTarget) {
	if (stopTypes === void 0) stopTypes = [];
	for (var i$1 = 0; i$1 < path.length; i$1++) {
		var group = path[i$1];
		if (blockGroupTypes.indexOf(group.blockGroupType) >= 0 && (!isValidTarget || isValidTarget(group))) return i$1;
		else if (stopTypes.indexOf(group.blockGroupType) >= 0) return -1;
	}
	return -1;
}
function getSelectedSegmentsAndParagraphs(model, includingFormatHolder, includingEntity, mutate) {
	var selections = collectSelections(model, { includeListFormatHolder: includingFormatHolder ? "allSegments" : "never" });
	var result = [];
	selections.forEach(function(_a$5) {
		var segments = _a$5.segments, block = _a$5.block, path = _a$5.path;
		if (segments) {
			if (includingFormatHolder && !block && segments.length == 1 && path[0].blockGroupType == "ListItem" && segments[0] == path[0].formatHolder) {
				var list = path[0];
				if (mutate) mutateBlock(list);
				result.push([
					list.formatHolder,
					null,
					path
				]);
			} else if ((block === null || block === void 0 ? void 0 : block.blockType) == "Paragraph") {
				if (mutate) mutateBlock(block);
				segments.forEach(function(segment) {
					if (includingEntity || segment.segmentType != "Entity" || !segment.entityFormat.isReadonly) result.push([
						segment,
						block,
						path
					]);
				});
			}
		} else if ((block === null || block === void 0 ? void 0 : block.blockType) == "Entity" && includingEntity) result.push([
			block,
			null,
			path
		]);
	});
	return result;
}
function getSelectedSegments(model, includingFormatHolder, mutate) {
	return (mutate ? getSelectedSegmentsAndParagraphs(model, includingFormatHolder, false, true) : getSelectedSegmentsAndParagraphs(model, includingFormatHolder)).map(function(x$1) {
		return x$1[0];
	});
}
function getSelectedParagraphs(model, mutate) {
	var selections = collectSelections(model, { includeListFormatHolder: "never" });
	var result = [];
	removeUnmeaningfulSelections(selections);
	selections.forEach(function(_a$5) {
		var block = _a$5.block;
		if ((block === null || block === void 0 ? void 0 : block.blockType) == "Paragraph") result.push(mutate ? mutateBlock(block) : block);
	});
	return result;
}
function getOperationalBlocks(group, blockGroupTypes, stopTypes, deepFirst, isValidTarget) {
	var result = [];
	var findSequence = deepFirst ? blockGroupTypes.map(function(type) {
		return [type];
	}) : [blockGroupTypes];
	var selections = collectSelections(group, {
		includeListFormatHolder: "never",
		contentUnderSelectedTableCell: "ignoreForTable"
	});
	removeUnmeaningfulSelections(selections);
	selections.forEach(function(_a$5) {
		var path = _a$5.path, block = _a$5.block;
		var _loop_1 = function(i$2) {
			var groupIndex = getClosestAncestorBlockGroupIndex(path, findSequence[i$2], stopTypes, isValidTarget);
			if (groupIndex >= 0) {
				if (result.filter(function(x$1) {
					return x$1.block == path[groupIndex];
				}).length <= 0) result.push({
					parent: path[groupIndex + 1],
					block: path[groupIndex],
					path: path.slice(groupIndex + 1)
				});
				return "break";
			} else if (i$2 == findSequence.length - 1 && block) {
				result.push({
					parent: path[0],
					block,
					path
				});
				return "break";
			}
		};
		for (var i$1 = 0; i$1 < findSequence.length; i$1++) if (_loop_1(i$1) === "break") break;
	});
	return result;
}
function getFirstSelectedTable(model) {
	var selections = collectSelections(model, { includeListFormatHolder: "never" });
	var table;
	var resultPath = [];
	removeUnmeaningfulSelections(selections);
	selections.forEach(function(_a$5) {
		var block = _a$5.block, tableContext = _a$5.tableContext, path = _a$5.path;
		if (!table) {
			if ((block === null || block === void 0 ? void 0 : block.blockType) == "Table") {
				table = block;
				resultPath = __spreadArray([], __read(path), false);
			} else if (tableContext === null || tableContext === void 0 ? void 0 : tableContext.table) {
				table = tableContext.table;
				var parent_1 = path.filter(function(group) {
					return group.blocks.indexOf(tableContext.table) >= 0;
				})[0];
				var index$1 = path.indexOf(parent_1);
				resultPath = index$1 >= 0 ? path.slice(index$1) : [];
			}
		}
	});
	return [table, resultPath];
}
function getFirstSelectedListItem(model) {
	var listItem;
	getOperationalBlocks(model, ["ListItem"], ["TableCell"]).forEach(function(r$1) {
		if (!listItem && isBlockGroupOfType(r$1.block, "ListItem")) listItem = r$1.block;
	});
	return listItem;
}
function collectSelections(group, option) {
	var selections = [];
	iterateSelections(group, function(path, tableContext, block, segments) {
		selections.push({
			path,
			tableContext,
			block,
			segments
		});
	}, option);
	return selections;
}
function removeUnmeaningfulSelections(selections) {
	if (selections.length > 1 && isOnlySelectionMarkerSelected$1(selections, false)) selections.pop();
	if (selections.length > 1 && isOnlySelectionMarkerSelected$1(selections, true)) selections.shift();
}
function isOnlySelectionMarkerSelected$1(selections, checkFirstParagraph) {
	var _a$5;
	var selection = selections[checkFirstParagraph ? 0 : selections.length - 1];
	if (((_a$5 = selection.block) === null || _a$5 === void 0 ? void 0 : _a$5.blockType) == "Paragraph" && selection.segments && selection.segments.length > 0) {
		var allSegments = selection.block.segments;
		var segment = selection.segments[0];
		return selection.segments.length == 1 && segment.segmentType == "SelectionMarker" && segment == allSegments[checkFirstParagraph ? allSegments.length - 1 : 0];
	} else return false;
}
function hasSelectionInSegment(segment) {
	return segment.isSelected || segment.segmentType == "General" && segment.blocks.some(hasSelectionInBlock);
}
function hasSelectionInBlock(block) {
	switch (block.blockType) {
		case "Paragraph": return block.segments.some(hasSelectionInSegment);
		case "Table": return block.rows.some(function(row) {
			return row.cells.some(hasSelectionInBlockGroup);
		});
		case "BlockGroup": return hasSelectionInBlockGroup(block);
		case "Divider":
		case "Entity": return !!block.isSelected;
		default: return false;
	}
}
function hasSelectionInBlockGroup(group) {
	if (group.blockGroupType == "TableCell" && group.isSelected) return true;
	if (group.blocks.some(hasSelectionInBlock)) return true;
	return false;
}
function getSelectedCells(table) {
	var firstRow = -1;
	var firstColumn = -1;
	var lastRow = -1;
	var lastColumn = -1;
	var hasSelection = false;
	table.rows.forEach(function(row, rowIndex) {
		return row.cells.forEach(function(cell, colIndex) {
			if (hasSelectionInBlockGroup(cell)) {
				hasSelection = true;
				if (firstRow < 0) firstRow = rowIndex;
				if (firstColumn < 0) firstColumn = colIndex;
				lastRow = Math.max(lastRow, rowIndex);
				lastColumn = Math.max(lastColumn, colIndex);
			}
		});
	});
	return hasSelection ? {
		firstRow,
		firstColumn,
		lastRow,
		lastColumn
	} : null;
}
function setSelection(group, start, end) {
	setSelectionToBlockGroup(group, false, start || null, end || null);
}
function setSelectionToBlockGroup(group, isInSelection, start, end) {
	return handleSelection(isInSelection, group, start, end, function(isInSelection$1) {
		if (isGeneralSegment(group) && needToSetSelection(group, isInSelection$1)) setIsSelected(mutateBlock(group), isInSelection$1);
		var blocksToDelete = [];
		group.blocks.forEach(function(block, i$1) {
			isInSelection$1 = setSelectionToBlock(block, isInSelection$1, start, end);
			if (block.blockType == "Paragraph" && block.segments.length == 0 && block.isImplicit) blocksToDelete.push(i$1);
		});
		var index$1;
		if (blocksToDelete.length > 0) {
			var mutableGroup = mutateBlock(group);
			while ((index$1 = blocksToDelete.pop()) !== void 0) if (index$1 >= 0) mutableGroup.blocks.splice(index$1, 1);
		}
		return isInSelection$1;
	});
}
function setSelectionToBlock(block, isInSelection, start, end) {
	switch (block.blockType) {
		case "BlockGroup": return setSelectionToBlockGroup(block, isInSelection, start, end);
		case "Table": return setSelectionToTable(block, isInSelection, start, end);
		case "Divider":
		case "Entity": return handleSelection(isInSelection, block, start, end, function(isInSelection$1) {
			if (needToSetSelection(block, isInSelection$1)) {
				var mutableBlock = mutateBlock(block);
				if (isInSelection$1) mutableBlock.isSelected = true;
				else delete mutableBlock.isSelected;
			}
			return isInSelection$1;
		});
		case "Paragraph":
			var segmentsToDelete_1 = [];
			block.segments.forEach(function(segment, i$1) {
				isInSelection = handleSelection(isInSelection, segment, start, end, function(isInSelection$1) {
					return setSelectionToSegment(block, segment, isInSelection$1, segmentsToDelete_1, start, end, i$1);
				});
			});
			if (segmentsToDelete_1.length > 0) {
				var mutablePara = mutateBlock(block);
				var index$1 = void 0;
				while ((index$1 = segmentsToDelete_1.pop()) !== void 0) if (index$1 >= 0) mutablePara.segments.splice(index$1, 1);
			}
			return isInSelection;
		default: return isInSelection;
	}
}
function setSelectionToTable(table, isInSelection, start, end) {
	var first = findCell(table, start);
	var last = end ? findCell(table, end) : first;
	if (!isInSelection) for (var row = 0; row < table.rows.length; row++) {
		var currentRow = table.rows[row];
		for (var col = 0; col < currentRow.cells.length; col++) {
			var currentCell = table.rows[row].cells[col];
			var isSelected$1 = row >= first.row && row <= last.row && col >= first.col && col <= last.col;
			if (needToSetSelection(currentCell, isSelected$1)) setIsSelected(mutateBlock(currentCell), isSelected$1);
			if (!isSelected$1) setSelectionToBlockGroup(currentCell, false, start, end);
		}
	}
	else table.rows.forEach(function(row$1) {
		return row$1.cells.forEach(function(cell) {
			var wasInSelection = isInSelection;
			isInSelection = setSelectionToBlockGroup(cell, isInSelection, start, end);
			if (wasInSelection && isInSelection) mutateBlock(cell).isSelected = true;
		});
	});
	return isInSelection;
}
function findCell(table, cell) {
	var col = -1;
	return {
		row: cell ? table.rows.findIndex(function(row) {
			return (col = row.cells.indexOf(cell)) >= 0;
		}) : -1,
		col
	};
}
function setSelectionToSegment(paragraph, segment, isInSelection, segmentsToDelete, start, end, i$1) {
	switch (segment.segmentType) {
		case "SelectionMarker":
			if (!isInSelection || segment != start && segment != end) segmentsToDelete.push(i$1);
			return isInSelection;
		case "General":
			internalSetSelectionToSegment(paragraph, segment, isInSelection);
			return segment != start && segment != end ? setSelectionToBlockGroup(segment, isInSelection, start, end) : isInSelection;
		case "Image":
			var isSelectedAsImageSelection_1 = start == segment && (!end || end == segment);
			internalSetSelectionToSegment(paragraph, segment, isInSelection, !segment.isSelectedAsImageSelection != !isSelectedAsImageSelection_1 ? function(image) {
				return image.isSelectedAsImageSelection = isSelectedAsImageSelection_1;
			} : void 0);
			return isInSelection;
		default:
			internalSetSelectionToSegment(paragraph, segment, isInSelection);
			return isInSelection;
	}
}
function internalSetSelectionToSegment(paragraph, segment, isInSelection, additionAction) {
	if (additionAction || needToSetSelection(segment, isInSelection)) mutateSegment(paragraph, segment, function(mutableSegment) {
		setIsSelected(mutableSegment, isInSelection);
		additionAction === null || additionAction === void 0 || additionAction(mutableSegment);
	});
}
function needToSetSelection(selectable, isSelected$1) {
	return !selectable.isSelected != !isSelected$1;
}
function setIsSelected(selectable, value) {
	if (value) selectable.isSelected = true;
	else delete selectable.isSelected;
	return value;
}
function handleSelection(isInSelection, model, start, end, callback) {
	isInSelection = isInSelection || model == start;
	isInSelection = callback(isInSelection);
	return isInSelection && !!end && model != end;
}
function cloneModel(model, options) {
	var newModel = cloneBlockGroupBase(model, options || {});
	if (model.format) newModel.format = Object.assign({}, model.format);
	return newModel;
}
function cloneBlock(block, options) {
	switch (block.blockType) {
		case "BlockGroup":
			switch (block.blockGroupType) {
				case "FormatContainer": return cloneFormatContainer(block, options);
				case "General": return cloneGeneralBlock(block, options);
				case "ListItem": return cloneListItem(block, options);
			}
			break;
		case "Divider": return cloneDivider(block, options);
		case "Entity": return cloneEntity(block, options);
		case "Paragraph": return cloneParagraph(block, options);
		case "Table": return cloneTable(block, options);
	}
}
function cloneSegment(segment, options) {
	switch (segment.segmentType) {
		case "Br": return cloneSegmentBase(segment);
		case "Entity": return cloneEntity(segment, options);
		case "General": return cloneGeneralSegment(segment, options);
		case "Image": return cloneImage$1(segment);
		case "SelectionMarker": return cloneSelectionMarker(segment);
		case "Text": return cloneText(segment);
	}
}
function cloneModelWithFormat(model) {
	return { format: Object.assign({}, model.format) };
}
function cloneModelWithDataset(model) {
	return { dataset: Object.assign({}, model.dataset) };
}
function cloneBlockBase(block) {
	var blockType = block.blockType;
	return Object.assign({ blockType }, cloneModelWithFormat(block));
}
function cloneBlockGroupBase(group, options) {
	return {
		blockGroupType: group.blockGroupType,
		blocks: group.blocks.map(function(block) {
			return cloneBlock(block, options);
		})
	};
}
function cloneSegmentBase(segment) {
	var segmentType = segment.segmentType, isSelected$1 = segment.isSelected, code = segment.code, link = segment.link;
	var newSegment = Object.assign({
		segmentType,
		isSelected: isSelected$1
	}, cloneModelWithFormat(segment));
	if (code) newSegment.code = cloneModelWithFormat(code);
	if (link) newSegment.link = Object.assign(cloneModelWithFormat(link), cloneModelWithDataset(link));
	return newSegment;
}
function cloneEntity(entity, options) {
	var wrapper = entity.wrapper, entityFormat = entity.entityFormat;
	return Object.assign({
		wrapper: handleCachedElement(wrapper, "entity", options),
		entityFormat: __assign({}, entityFormat)
	}, cloneBlockBase(entity), cloneSegmentBase(entity));
}
function cloneParagraph(paragraph, options) {
	var cachedElement = paragraph.cachedElement, segments = paragraph.segments, isImplicit = paragraph.isImplicit, decorator = paragraph.decorator, segmentFormat = paragraph.segmentFormat;
	var newParagraph = Object.assign({
		cachedElement: handleCachedElement(cachedElement, "cache", options),
		isImplicit,
		segments: segments.map(function(segment) {
			return cloneSegment(segment, options);
		}),
		segmentFormat: segmentFormat ? __assign({}, segmentFormat) : void 0
	}, cloneBlockBase(paragraph), cloneModelWithFormat(paragraph));
	if (decorator) newParagraph.decorator = Object.assign({ tagName: decorator.tagName }, cloneModelWithFormat(decorator));
	return newParagraph;
}
function cloneTable(table, options) {
	var cachedElement = table.cachedElement, widths = table.widths, rows = table.rows;
	return Object.assign({
		cachedElement: handleCachedElement(cachedElement, "cache", options),
		widths: Array.from(widths),
		rows: rows.map(function(row) {
			return cloneTableRow(row, options);
		})
	}, cloneBlockBase(table), cloneModelWithDataset(table));
}
function cloneTableRow(row, options) {
	var height = row.height, cells = row.cells, cachedElement = row.cachedElement;
	return Object.assign({
		height,
		cachedElement: handleCachedElement(cachedElement, "cache", options),
		cells: cells.map(function(cell) {
			return cloneTableCell(cell, options);
		})
	}, cloneModelWithFormat(row));
}
function cloneTableCell(cell, options) {
	var cachedElement = cell.cachedElement, isSelected$1 = cell.isSelected, spanAbove = cell.spanAbove, spanLeft = cell.spanLeft, isHeader = cell.isHeader;
	return Object.assign({
		cachedElement: handleCachedElement(cachedElement, "cache", options),
		isSelected: isSelected$1,
		spanAbove,
		spanLeft,
		isHeader
	}, cloneBlockGroupBase(cell, options), cloneModelWithFormat(cell), cloneModelWithDataset(cell));
}
function cloneFormatContainer(container, options) {
	var tagName = container.tagName, cachedElement = container.cachedElement;
	var newContainer = Object.assign({
		tagName,
		cachedElement: handleCachedElement(cachedElement, "cache", options)
	}, cloneBlockBase(container), cloneBlockGroupBase(container, options));
	if (container.zeroFontSize) newContainer.zeroFontSize = true;
	return newContainer;
}
function cloneListItem(item, options) {
	var formatHolder = item.formatHolder, levels = item.levels, cachedElement = item.cachedElement;
	return Object.assign({
		formatHolder: cloneSelectionMarker(formatHolder),
		levels: levels.map(cloneListLevel),
		cachedElement: handleCachedElement(cachedElement, "cache", options)
	}, cloneBlockBase(item), cloneBlockGroupBase(item, options));
}
function cloneListLevel(level) {
	var listType = level.listType;
	return Object.assign({ listType }, cloneModelWithFormat(level), cloneModelWithDataset(level));
}
function cloneDivider(divider, options) {
	var tagName = divider.tagName, isSelected$1 = divider.isSelected, cachedElement = divider.cachedElement;
	return Object.assign({
		isSelected: isSelected$1,
		tagName,
		cachedElement: handleCachedElement(cachedElement, "cache", options)
	}, cloneBlockBase(divider));
}
function cloneGeneralBlock(general, options) {
	var element = general.element;
	return Object.assign({ element: handleCachedElement(element, "general", options) }, cloneBlockBase(general), cloneBlockGroupBase(general, options));
}
function cloneSelectionMarker(marker) {
	return Object.assign({ isSelected: marker.isSelected }, cloneSegmentBase(marker));
}
function cloneImage$1(image) {
	var src = image.src, alt = image.alt, title = image.title, isSelectedAsImageSelection = image.isSelectedAsImageSelection;
	return Object.assign({
		src,
		alt,
		title,
		isSelectedAsImageSelection
	}, cloneSegmentBase(image), cloneModelWithDataset(image));
}
function cloneGeneralSegment(general, options) {
	return Object.assign(cloneGeneralBlock(general, options), cloneSegmentBase(general));
}
function cloneText(textSegment) {
	var text = textSegment.text;
	return Object.assign({ text }, cloneSegmentBase(textSegment));
}
function handleCachedElement(node, type, options) {
	var includeCachedElement = options.includeCachedElement;
	if (!node) return;
	else if (!includeCachedElement) return type == "cache" ? void 0 : node.cloneNode(true);
	else if (includeCachedElement === true) return node;
	else {
		var result = includeCachedElement(node, type);
		if ((type == "general" || type == "entity") && !result) throw new Error("Entity and General Model must has wrapper element");
		return result;
	}
}
function createNumberDefinition(isOptional, value, minValue, maxValue, allowNull) {
	return {
		type: "number",
		isOptional,
		value,
		maxValue,
		minValue,
		allowNull
	};
}
function createBooleanDefinition(isOptional, value, allowNull) {
	return {
		type: "boolean",
		isOptional,
		value,
		allowNull
	};
}
function createStringDefinition(isOptional, value, allowNull) {
	return {
		type: "string",
		isOptional,
		value,
		allowNull
	};
}
function createObjectDefinition(propertyDef, isOptional, allowNull) {
	return {
		type: "object",
		isOptional,
		propertyDef,
		allowNull
	};
}
var TableCellMetadataFormatDefinition = createObjectDefinition({
	bgColorOverride: createBooleanDefinition(true),
	vAlignOverride: createBooleanDefinition(true),
	borderOverride: createBooleanDefinition(true)
}, false, true);
function updateTableCellMetadata(cell, callback) {
	return updateMetadata(cell, callback, TableCellMetadataFormatDefinition);
}
var DARK_COLORS_LIGHTNESS = 20;
var BRIGHT_COLORS_LIGHTNESS = 80;
var White = "#ffffff";
var Black = "#000000";
function setTableCellBackgroundColor(cell, color, isColorOverride, applyToSegments) {
	if (color) {
		cell.format.backgroundColor = color;
		if (isColorOverride) updateTableCellMetadata(cell, function(metadata) {
			metadata = metadata || {};
			metadata.bgColorOverride = true;
			return metadata;
		});
		var lightness = calculateLightness(color);
		if (lightness < DARK_COLORS_LIGHTNESS) cell.format.textColor = White;
		else if (lightness > BRIGHT_COLORS_LIGHTNESS) cell.format.textColor = Black;
		else delete cell.format.textColor;
		if (applyToSegments) setAdaptiveCellColor(cell);
	} else {
		delete cell.format.backgroundColor;
		delete cell.format.textColor;
		if (applyToSegments) removeAdaptiveCellColor(cell);
	}
}
function removeAdaptiveCellColor(cell) {
	cell.blocks.forEach(function(readonlyBlock) {
		var _a$5, _b$1;
		if (readonlyBlock.blockType == "Paragraph") {
			var block = mutateBlock(readonlyBlock);
			if (((_a$5 = block.segmentFormat) === null || _a$5 === void 0 ? void 0 : _a$5.textColor) && shouldRemoveColor((_b$1 = block.segmentFormat) === null || _b$1 === void 0 ? void 0 : _b$1.textColor, cell.format.backgroundColor || "")) delete block.segmentFormat.textColor;
			block.segments.forEach(function(segment) {
				if (segment.format.textColor && shouldRemoveColor(segment.format.textColor, cell.format.backgroundColor || "")) delete segment.format.textColor;
			});
		}
	});
}
function setAdaptiveCellColor(cell) {
	if (cell.format.textColor) cell.blocks.forEach(function(readonlyBlock) {
		var _a$5;
		if (readonlyBlock.blockType == "Paragraph") {
			var block = mutateBlock(readonlyBlock);
			if (!((_a$5 = block.segmentFormat) === null || _a$5 === void 0 ? void 0 : _a$5.textColor)) block.segmentFormat = __assign(__assign({}, block.segmentFormat), { textColor: cell.format.textColor });
			block.segments.forEach(function(segment) {
				var _a$6;
				if (!((_a$6 = segment.format) === null || _a$6 === void 0 ? void 0 : _a$6.textColor)) segment.format = __assign(__assign({}, segment.format), { textColor: cell.format.textColor });
			});
		}
	});
}
function shouldRemoveColor(textColor, cellBackgroundColor) {
	var lightness = calculateLightness(cellBackgroundColor);
	if ([White, "rgb(255,255,255)"].indexOf(textColor) > -1 && (lightness > BRIGHT_COLORS_LIGHTNESS || cellBackgroundColor == "") || [Black, "rgb(0,0,0)"].indexOf(textColor) > -1 && (lightness < DARK_COLORS_LIGHTNESS || cellBackgroundColor == "")) return true;
	return false;
}
function calculateLightness(color) {
	var colorValues = parseColor(color);
	if (colorValues) {
		var red = colorValues[0] / 255;
		var green = colorValues[1] / 255;
		var blue = colorValues[2] / 255;
		return (Math.max(red, green, blue) + Math.min(red, green, blue)) * 50;
	} else return 255;
}
var TableBorderFormat = {
	Min: 0,
	Default: 0,
	ListWithSideBorders: 1,
	NoHeaderBorders: 2,
	NoSideBorders: 3,
	FirstColumnHeaderExternal: 4,
	EspecialType1: 5,
	EspecialType2: 6,
	EspecialType3: 7,
	Clear: 8,
	Max: 8
};
var NullStringDefinition = createStringDefinition(false, void 0, true);
var BooleanDefinition$1 = createBooleanDefinition(false);
var TableFormatDefinition = createObjectDefinition({
	topBorderColor: NullStringDefinition,
	bottomBorderColor: NullStringDefinition,
	verticalBorderColor: NullStringDefinition,
	hasHeaderRow: BooleanDefinition$1,
	headerRowColor: NullStringDefinition,
	hasFirstColumn: BooleanDefinition$1,
	hasBandedColumns: BooleanDefinition$1,
	hasBandedRows: BooleanDefinition$1,
	bgColorEven: NullStringDefinition,
	bgColorOdd: NullStringDefinition,
	tableBorderFormat: createNumberDefinition(false, void 0, TableBorderFormat.Min, TableBorderFormat.Max),
	verticalAlign: NullStringDefinition
}, false, true);
function getTableMetadata(table) {
	return getMetadata(table, TableFormatDefinition);
}
function updateTableMetadata(table, callback) {
	return updateMetadata(table, callback, TableFormatDefinition);
}
var _a$3;
var DEFAULT_FORMAT = {
	topBorderColor: "#ABABAB",
	bottomBorderColor: "#ABABAB",
	verticalBorderColor: "#ABABAB",
	hasHeaderRow: false,
	hasFirstColumn: false,
	hasBandedRows: false,
	hasBandedColumns: false,
	bgColorEven: null,
	bgColorOdd: "#ABABAB20",
	headerRowColor: "#ABABAB",
	tableBorderFormat: TableBorderFormat.Default,
	verticalAlign: null
};
function applyTableFormat(table, newFormat, keepCellShade) {
	var mutableTable = mutateBlock(table);
	var rows = mutableTable.rows;
	updateTableMetadata(mutableTable, function(format) {
		var effectiveMetadata = __assign(__assign(__assign({}, DEFAULT_FORMAT), format), newFormat);
		var metaOverrides = updateOverrides(rows, !keepCellShade);
		formatCells(rows, effectiveMetadata, metaOverrides);
		setFirstColumnFormatBorders(rows, effectiveMetadata);
		setHeaderRowFormat(rows, effectiveMetadata, metaOverrides);
		return effectiveMetadata;
	});
}
function updateOverrides(rows, removeCellShade) {
	var overrides = {
		bgColorOverrides: [],
		vAlignOverrides: [],
		borderOverrides: []
	};
	rows.forEach(function(row) {
		var bgColorOverrides = [];
		var vAlignOverrides = [];
		var borderOverrides = [];
		overrides.bgColorOverrides.push(bgColorOverrides);
		overrides.vAlignOverrides.push(vAlignOverrides);
		overrides.borderOverrides.push(borderOverrides);
		row.cells.forEach(function(cell) {
			updateTableCellMetadata(mutateBlock(cell), function(metadata) {
				if (metadata && removeCellShade) {
					bgColorOverrides.push(false);
					delete metadata.bgColorOverride;
				} else bgColorOverrides.push(!!(metadata === null || metadata === void 0 ? void 0 : metadata.bgColorOverride));
				vAlignOverrides.push(!!(metadata === null || metadata === void 0 ? void 0 : metadata.vAlignOverride));
				borderOverrides.push(!!(metadata === null || metadata === void 0 ? void 0 : metadata.borderOverride));
				return metadata;
			});
		});
	});
	return overrides;
}
var BorderFormatters = (_a$3 = {}, _a$3[TableBorderFormat.Default] = function(_) {
	return [
		false,
		false,
		false,
		false
	];
}, _a$3[TableBorderFormat.ListWithSideBorders] = function(_a$5) {
	var lastColumn = _a$5.lastColumn, firstColumn = _a$5.firstColumn;
	return [
		false,
		!lastColumn,
		false,
		!firstColumn
	];
}, _a$3[TableBorderFormat.FirstColumnHeaderExternal] = function(_a$5) {
	var firstColumn = _a$5.firstColumn, firstRow = _a$5.firstRow, lastColumn = _a$5.lastColumn, lastRow = _a$5.lastRow;
	return [
		!firstRow,
		!lastColumn && !firstColumn || firstColumn && firstRow,
		!lastRow && !firstRow,
		!firstColumn
	];
}, _a$3[TableBorderFormat.NoHeaderBorders] = function(_a$5) {
	var firstRow = _a$5.firstRow, firstColumn = _a$5.firstColumn, lastColumn = _a$5.lastColumn;
	return [
		firstRow,
		firstRow || lastColumn,
		false,
		firstRow || firstColumn
	];
}, _a$3[TableBorderFormat.NoSideBorders] = function(_a$5) {
	var firstColumn = _a$5.firstColumn;
	return [
		false,
		_a$5.lastColumn,
		false,
		firstColumn
	];
}, _a$3[TableBorderFormat.EspecialType1] = function(_a$5) {
	var firstRow = _a$5.firstRow, firstColumn = _a$5.firstColumn;
	return [
		firstColumn && !firstRow,
		firstRow,
		firstColumn && !firstRow,
		firstRow && !firstColumn
	];
}, _a$3[TableBorderFormat.EspecialType2] = function(_a$5) {
	var firstRow = _a$5.firstRow, firstColumn = _a$5.firstColumn;
	return [
		!firstRow,
		firstRow || !firstColumn,
		!firstRow,
		!firstColumn
	];
}, _a$3[TableBorderFormat.EspecialType3] = function(_a$5) {
	var firstColumn = _a$5.firstColumn, firstRow = _a$5.firstRow;
	return [
		true,
		firstRow || !firstColumn,
		!firstRow,
		true
	];
}, _a$3[TableBorderFormat.Clear] = function() {
	return [
		true,
		true,
		true,
		true
	];
}, _a$3);
function formatCells(rows, format, metaOverrides) {
	var hasBandedRows = format.hasBandedRows, hasBandedColumns = format.hasBandedColumns, bgColorOdd = format.bgColorOdd, bgColorEven = format.bgColorEven, hasFirstColumn = format.hasFirstColumn;
	rows.forEach(function(row, rowIndex) {
		row.cells.forEach(function(readonlyCell, colIndex) {
			var _a$5;
			var cell = mutateBlock(readonlyCell);
			if (!metaOverrides.borderOverrides[rowIndex][colIndex] && typeof format.tableBorderFormat == "number") {
				var transparentBorderMatrix = (_a$5 = BorderFormatters[format.tableBorderFormat]) === null || _a$5 === void 0 ? void 0 : _a$5.call(BorderFormatters, {
					firstRow: rowIndex === 0,
					lastRow: rowIndex === rows.length - 1,
					firstColumn: colIndex === 0,
					lastColumn: colIndex === row.cells.length - 1
				});
				var formatColor_1 = [
					format.topBorderColor,
					format.verticalBorderColor,
					format.bottomBorderColor,
					format.verticalBorderColor
				];
				transparentBorderMatrix === null || transparentBorderMatrix === void 0 || transparentBorderMatrix.forEach(function(alwaysUseTransparent, i$1) {
					var borderColor = !alwaysUseTransparent && formatColor_1[i$1] || "";
					cell.format[BorderKeys[i$1]] = combineBorderValue({
						style: getBorderStyleFromColor(borderColor),
						width: "1px",
						color: borderColor
					});
				});
			}
			if (!metaOverrides.bgColorOverrides[rowIndex][colIndex]) {
				var color = void 0;
				if (hasFirstColumn && colIndex == 0 && rowIndex > 0) color = null;
				else color = hasBandedRows || hasBandedColumns ? hasBandedColumns && colIndex % 2 != 0 || hasBandedRows && rowIndex % 2 != 0 ? bgColorOdd : bgColorEven : bgColorEven;
				setTableCellBackgroundColor(cell, color, false, true);
			}
			if (format.verticalAlign && !metaOverrides.vAlignOverrides[rowIndex][colIndex]) cell.format.verticalAlign = format.verticalAlign;
			cell.isHeader = false;
		});
	});
}
function setFirstColumnFormatBorders(rows, format) {
	if (!format.hasFirstColumn) return;
	rows.forEach(function(row, rowIndex) {
		row.cells.forEach(function(readonlyCell, cellIndex) {
			var cell = mutateBlock(readonlyCell);
			if (cellIndex === 0) {
				cell.isHeader = true;
				switch (rowIndex) {
					case 0:
						cell.isHeader = !!format.hasHeaderRow;
						if (cell.isHeader) cell.format.fontWeight = "bold";
						break;
					case rows.length - 1:
						setBorderColor(cell.format, "borderTop");
						break;
					case 1:
						setBorderColor(cell.format, "borderBottom");
						break;
					default:
						setBorderColor(cell.format, "borderTop");
						setBorderColor(cell.format, "borderBottom");
						break;
				}
			}
		});
	});
}
function setHeaderRowFormat(rows, format, metaOverrides) {
	var _a$5;
	if (!format.hasHeaderRow) return;
	var rowIndex = 0;
	(_a$5 = rows[rowIndex]) === null || _a$5 === void 0 || _a$5.cells.forEach(function(readonlyCell, cellIndex) {
		var cell = mutateBlock(readonlyCell);
		cell.isHeader = true;
		cell.format.fontWeight = "bold";
		if (format.headerRowColor) {
			if (!metaOverrides.bgColorOverrides[rowIndex][cellIndex]) setTableCellBackgroundColor(cell, format.headerRowColor, false, true);
			setBorderColor(cell.format, "borderTop", format.headerRowColor);
			setBorderColor(cell.format, "borderRight", format.headerRowColor);
			setBorderColor(cell.format, "borderLeft", format.headerRowColor);
		}
	});
}
function setBorderColor(format, key, value) {
	var border = extractBorderValues(format[key]);
	border.color = value || "";
	border.style = getBorderStyleFromColor(border.color);
	format[key] = combineBorderValue(border);
}
function getBorderStyleFromColor(color) {
	return !color || color == "transparent" ? "none" : "solid";
}
function deleteBlock(blocks, blockToDelete, replacement, context, direction) {
	var index$1 = blocks.indexOf(blockToDelete);
	switch (blockToDelete.blockType) {
		case "Table":
		case "Divider":
			replacement ? blocks.splice(index$1, 1, replacement) : blocks.splice(index$1, 1);
			return true;
		case "Entity":
			var operation = blockToDelete.isSelected ? "overwrite" : direction == "forward" ? "removeFromStart" : direction == "backward" ? "removeFromEnd" : void 0;
			if (operation !== void 0) {
				replacement ? blocks.splice(index$1, 1, replacement) : blocks.splice(index$1, 1);
				context === null || context === void 0 || context.deletedEntities.push({
					entity: blockToDelete,
					operation
				});
			}
			return true;
		case "BlockGroup": switch (blockToDelete.blockGroupType) {
			case "General": if (replacement) {
				blocks.splice(index$1, 1, replacement);
				return true;
			} else return false;
			case "ListItem":
			case "FormatContainer":
				blocks.splice(index$1, 1);
				return true;
		}
	}
	return false;
}
function deleteSingleChar(text, isForward) {
	var array = __spreadArray([], __read(text), false);
	var deleteLength = 0;
	for (var i$1 = isForward ? 0 : array.length - 1, deleteState = "notDeleted"; i$1 >= 0 && i$1 < array.length && deleteState != "done"; i$1 += isForward ? 1 : -1) switch (array[i$1]) {
		case "‍":
		case "⃣":
		case "︎":
		case "️":
			deleteState = "notDeleted";
			deleteLength++;
			break;
		default:
			if (deleteState == "notDeleted") {
				deleteState = "waiting";
				deleteLength++;
			} else if (deleteState == "waiting") deleteState = "done";
			break;
	}
	array.splice(isForward ? 0 : array.length - deleteLength, deleteLength);
	return array.join("");
}
function deleteSegment(readonlyParagraph, readonlySegmentToDelete, context, direction, undeletableSegments) {
	var _a$5 = __read(mutateSegment(readonlyParagraph, readonlySegmentToDelete), 3), paragraph = _a$5[0], segmentToDelete = _a$5[1], index$1 = _a$5[2];
	var segments = paragraph.segments;
	var preserveWhiteSpace = isWhiteSpacePreserved(paragraph.format.whiteSpace);
	var isForward = direction == "forward";
	var isBackward = direction == "backward";
	if (!preserveWhiteSpace) normalizePreviousSegment(paragraph, segments, index$1);
	switch (segmentToDelete === null || segmentToDelete === void 0 ? void 0 : segmentToDelete.segmentType) {
		case "Br":
		case "Image":
		case "SelectionMarker":
			removeSegment(segments, index$1, direction, undeletableSegments);
			return true;
		case "Entity":
			var operation = segmentToDelete.isSelected ? "overwrite" : isForward ? "removeFromStart" : isBackward ? "removeFromEnd" : void 0;
			if (operation !== void 0) {
				removeSegment(segments, index$1, direction, undeletableSegments);
				context === null || context === void 0 || context.deletedEntities.push({
					entity: segmentToDelete,
					operation
				});
			}
			return true;
		case "Text":
			if (segmentToDelete.text.length == 0 || segmentToDelete.isSelected) {
				segmentToDelete.text = "";
				removeSegment(segments, index$1, direction, undeletableSegments);
			} else if (direction) {
				var text = segmentToDelete.text;
				text = deleteSingleChar(text, isForward);
				if (!preserveWhiteSpace) text = normalizeText(text, isForward);
				segmentToDelete.text = text;
				if (text == "") removeSegment(segments, index$1, direction, undeletableSegments);
			}
			return true;
		case "General": if (segmentToDelete.isSelected) {
			removeSegment(segments, index$1, direction, undeletableSegments);
			return true;
		} else return false;
		default: return false;
	}
}
function removeSegment(segments, index$1, direction, undeletableSegments) {
	var _a$5;
	var segment = segments.splice(index$1, 1)[0];
	if ((_a$5 = segment.link) === null || _a$5 === void 0 ? void 0 : _a$5.format.undeletable) {
		delete segment.isSelected;
		if (undeletableSegments) undeletableSegments.push(segment);
		else {
			var insertIndex = void 0;
			switch (direction) {
				case "forward":
					insertIndex = index$1 > 0 && segments[index$1 - 1].segmentType == "SelectionMarker" ? index$1 - 1 : index$1;
					break;
				case "backward":
					insertIndex = index$1 < segments.length && segments[index$1].segmentType == "SelectionMarker" ? index$1 + 1 : index$1;
					break;
				default: insertIndex = index$1;
			}
			segments.splice(insertIndex, 0, segment);
		}
	}
}
function normalizePreviousSegment(paragraph, segments, currentIndex) {
	var _a$5;
	var index$1 = currentIndex - 1;
	while (((_a$5 = segments[index$1]) === null || _a$5 === void 0 ? void 0 : _a$5.segmentType) == "SelectionMarker") index$1--;
	var segment = segments[index$1];
	if (segment) normalizeSingleSegment(paragraph, segment);
}
function getSegmentTextFormat(segment, includingBIU) {
	var _a$5;
	var format = (_a$5 = segment.format) !== null && _a$5 !== void 0 ? _a$5 : {};
	return removeUndefinedValues({
		fontFamily: format.fontFamily,
		fontSize: format.fontSize,
		textColor: format.textColor,
		backgroundColor: format.backgroundColor,
		letterSpacing: format.letterSpacing,
		lineHeight: format.lineHeight,
		fontWeight: includingBIU ? format.fontWeight : void 0,
		italic: includingBIU ? format.italic : void 0,
		underline: includingBIU ? format.underline : void 0
	});
}
var removeUndefinedValues = function(format) {
	var textFormat = {};
	Object.keys(format).filter(function(key) {
		var value = format[key];
		if (value !== void 0) textFormat[key] = value;
	});
	return textFormat;
};
var DeleteSelectionIteratingOptions = {
	contentUnderSelectedTableCell: "ignoreForTableOrCell",
	contentUnderSelectedGeneralElement: "generalElementOnly",
	includeListFormatHolder: "never"
};
function deleteExpandedSelection(model, formatContext) {
	var context = {
		deleteResult: "notDeleted",
		insertPoint: null,
		formatContext,
		undeletableSegments: []
	};
	iterateSelections(model, function(path, tableContext, readonlyBlock, readonlySegments) {
		var paragraph = createParagraph(true, void 0, model.format);
		var markerFormat = model.format;
		var insertMarkerIndex = 0;
		if (readonlySegments && (readonlyBlock === null || readonlyBlock === void 0 ? void 0 : readonlyBlock.blockType) == "Paragraph") {
			var _a$5 = __read(mutateSegments(readonlyBlock, readonlySegments), 3), block_1 = _a$5[0], segments = _a$5[1], indexes = _a$5[2];
			if (segments[0]) {
				paragraph = block_1;
				insertMarkerIndex = indexes[0];
				markerFormat = getSegmentTextFormat(segments[0], true);
				var isFirstDeletingParagraph_1 = !context.lastParagraph;
				context.lastParagraph = paragraph;
				context.lastTableContext = tableContext;
				segments.forEach(function(segment, i$1) {
					if (i$1 == 0 && !context.insertPoint && segment.segmentType == "SelectionMarker") context.insertPoint = createInsertPoint(segment, block_1, path, tableContext);
					else if (deleteSegment(block_1, segment, context.formatContext, void 0, isFirstDeletingParagraph_1 ? void 0 : context.undeletableSegments)) context.deleteResult = "range";
				});
				if (context.deleteResult == "range") setParagraphNotImplicit(block_1);
			}
		} else if (readonlyBlock) {
			var blocks = mutateBlock(path[0]).blocks;
			if (deleteBlock(blocks, readonlyBlock, paragraph, context.formatContext)) context.deleteResult = "range";
		} else if (tableContext) {
			var table = tableContext.table, colIndex = tableContext.colIndex, rowIndex = tableContext.rowIndex;
			var row = mutateBlock(table).rows[rowIndex];
			var cell = mutateBlock(row.cells[colIndex]);
			path = __spreadArray([cell], __read(path), false);
			paragraph.segments.push(createBr(model.format));
			cell.blocks = [paragraph];
			context.deleteResult = "range";
		}
		if (!context.insertPoint) {
			var marker = createSelectionMarker(markerFormat);
			setParagraphNotImplicit(paragraph);
			paragraph.segments.splice(insertMarkerIndex, 0, marker);
			context.insertPoint = createInsertPoint(marker, paragraph, path, tableContext);
		}
	}, DeleteSelectionIteratingOptions);
	return context;
}
function createInsertPoint(marker, paragraph, path, tableContext) {
	return {
		marker,
		paragraph,
		path,
		tableContext
	};
}
function runEditSteps(steps, context) {
	steps.forEach(function(step) {
		if (step && isValidDeleteSelectionContext(context)) step(context);
	});
}
function isValidDeleteSelectionContext(context) {
	return !!context.insertPoint;
}
function deleteSelection(model, additionalSteps, formatContext) {
	if (additionalSteps === void 0) additionalSteps = [];
	var context = deleteExpandedSelection(model, formatContext);
	runEditSteps(additionalSteps.filter(function(x$1) {
		return !!x$1;
	}), context);
	mergeParagraphAfterDelete(context);
	return context;
}
function mergeParagraphAfterDelete(context) {
	var _a$5, _b$1;
	var insertPoint = context.insertPoint, deleteResult = context.deleteResult, lastParagraph = context.lastParagraph, lastTableContext = context.lastTableContext, undeletableSegments = context.undeletableSegments;
	if (insertPoint && deleteResult != "notDeleted" && deleteResult != "nothingToDelete" && lastParagraph && lastParagraph != insertPoint.paragraph && lastTableContext == insertPoint.tableContext) {
		var mutableLastParagraph = mutateBlock(lastParagraph);
		var mutableInsertingParagraph = mutateBlock(insertPoint.paragraph);
		if (undeletableSegments) (_a$5 = mutableLastParagraph.segments).unshift.apply(_a$5, __spreadArray([], __read(undeletableSegments), false));
		(_b$1 = mutableInsertingParagraph.segments).push.apply(_b$1, __spreadArray([], __read(mutableLastParagraph.segments), false));
		mutableLastParagraph.segments = [];
	}
}
var EmptySegmentFormat = {
	backgroundColor: "",
	fontFamily: "",
	fontSize: "",
	fontWeight: "",
	italic: false,
	letterSpacing: "",
	lineHeight: "",
	strikethrough: false,
	superOrSubScriptSequence: "",
	textColor: "",
	underline: false
};
function normalizeTable$1(readonlyTable, defaultSegmentFormat) {
	var _a$5;
	var table = mutateBlock(readonlyTable);
	var format = table.format;
	if (!format.borderCollapse || !format.useBorderBox) {
		format.borderCollapse = true;
		format.useBorderBox = true;
	}
	table.rows.forEach(function(row, rowIndex$1) {
		row.cells.forEach(function(readonlyCell, colIndex$1) {
			var cell = mutateBlock(readonlyCell);
			if (cell.blocks.length == 0) {
				var format_1 = cell.format.textColor ? __assign(__assign({}, defaultSegmentFormat), { textColor: cell.format.textColor }) : defaultSegmentFormat;
				addBlock(cell, createParagraph(void 0, void 0, format_1));
				addSegment(cell, createBr(format_1));
			}
			if (rowIndex$1 == 0) cell.spanAbove = false;
			else if (rowIndex$1 > 0 && colIndex$1 > 0 && cell.isHeader) cell.isHeader = false;
			if (colIndex$1 == 0) cell.spanLeft = false;
			cell.format.useBorderBox = true;
		});
		if (row.height < 22) row.height = 22;
	});
	var columns = Math.max.apply(Math, __spreadArray([], __read(table.rows.map(function(row) {
		return row.cells.length;
	})), false));
	for (var i$1 = 0; i$1 < columns; i$1++) if (table.widths[i$1] === void 0) table.widths[i$1] = getTableCellWidth(columns);
	else if (table.widths[i$1] < 30) table.widths[i$1] = 30;
	var colCount = ((_a$5 = table.rows[0]) === null || _a$5 === void 0 ? void 0 : _a$5.cells.length) || 0;
	var _loop_1 = function(colIndex$1) {
		table.rows.forEach(function(row) {
			var cell = row.cells[colIndex$1];
			var leftCell = row.cells[colIndex$1 - 1];
			if (cell && leftCell && cell.spanLeft) tryMoveBlocks(leftCell, cell);
		});
		if (table.rows.every(function(row) {
			var _a$6;
			return (_a$6 = row.cells[colIndex$1]) === null || _a$6 === void 0 ? void 0 : _a$6.spanLeft;
		})) {
			table.rows.forEach(function(row) {
				return row.cells.splice(colIndex$1, 1);
			});
			table.widths.splice(colIndex$1 - 1, 2, table.widths[colIndex$1 - 1] + table.widths[colIndex$1]);
		}
	};
	for (var colIndex = colCount - 1; colIndex > 0; colIndex--) _loop_1(colIndex);
	var _loop_2 = function(rowIndex$1) {
		var row = table.rows[rowIndex$1];
		row.cells.forEach(function(cell, colIndex$1) {
			var _a$6;
			var aboveCell = (_a$6 = table.rows[rowIndex$1 - 1]) === null || _a$6 === void 0 ? void 0 : _a$6.cells[colIndex$1];
			if (aboveCell && cell.spanAbove) tryMoveBlocks(aboveCell, cell);
		});
		if (row.cells.every(function(cell) {
			return cell.spanAbove;
		})) {
			table.rows[rowIndex$1 - 1].height += row.height;
			table.rows.splice(rowIndex$1, 1);
		}
	};
	for (var rowIndex = table.rows.length - 1; rowIndex > 0; rowIndex--) _loop_2(rowIndex);
}
function getTableCellWidth(columns) {
	if (columns <= 4) return 120;
	else if (columns <= 6) return 100;
	else return 70;
}
function tryMoveBlocks(targetCell, sourceCell) {
	var _a$5;
	if (!sourceCell.blocks.every(function(block) {
		return block.blockType == "Paragraph" && hasOnlyBrSegment(block.segments);
	})) {
		(_a$5 = mutateBlock(targetCell).blocks).push.apply(_a$5, __spreadArray([], __read(sourceCell.blocks), false));
		mutateBlock(sourceCell).blocks = [];
	}
}
function hasOnlyBrSegment(segments) {
	segments = segments.filter(function(s$1) {
		return s$1.segmentType != "SelectionMarker";
	});
	return segments.length == 0 || segments.length == 1 && segments[0].segmentType == "Br";
}
var HeadingTags = [
	"h1",
	"h2",
	"h3",
	"h4",
	"h5",
	"h6"
];
var KeysOfSegmentFormat = getObjectKeys(EmptySegmentFormat);
function mergeModel(target, source, context, options) {
	var _a$5;
	var insertPosition = (_a$5 = options === null || options === void 0 ? void 0 : options.insertPosition) !== null && _a$5 !== void 0 ? _a$5 : deleteSelection(target, [], context).insertPoint;
	var _b$1 = options || {}, addParagraphAfterMergedContent = _b$1.addParagraphAfterMergedContent, mergeFormat = _b$1.mergeFormat, mergeTable = _b$1.mergeTable;
	if (addParagraphAfterMergedContent && !mergeTable) {
		var _c$1 = insertPosition || {}, paragraph = _c$1.paragraph, marker = _c$1.marker;
		addBlock(source, createParagraph(false, paragraph === null || paragraph === void 0 ? void 0 : paragraph.format, marker === null || marker === void 0 ? void 0 : marker.format));
	}
	if (insertPosition) {
		if (mergeFormat && mergeFormat != "none") applyDefaultFormat$1(source, __assign(__assign({}, target.format || {}), insertPosition.marker.format), mergeFormat);
		for (var i$1 = 0; i$1 < source.blocks.length; i$1++) {
			var block = source.blocks[i$1];
			switch (block.blockType) {
				case "Paragraph":
					mergeParagraph(insertPosition, block, i$1 == 0, context, options);
					break;
				case "Divider":
					insertBlock(insertPosition, block);
					break;
				case "Entity":
					insertBlock(insertPosition, block);
					context === null || context === void 0 || context.newEntities.push(block);
					break;
				case "Table":
					if (source.blocks.length == 1 && mergeTable) mergeTables(insertPosition, block, source);
					else insertBlock(insertPosition, block);
					break;
				case "BlockGroup":
					switch (block.blockGroupType) {
						case "General":
						case "FormatContainer":
							insertBlock(insertPosition, block);
							break;
						case "ListItem":
							mergeList(insertPosition, block);
							break;
					}
					break;
			}
		}
	}
	normalizeContentModel(target);
	return insertPosition;
}
function mergeParagraph(markerPosition, newPara, mergeToCurrentParagraph, context, option) {
	var paragraph = markerPosition.paragraph, marker = markerPosition.marker;
	var newParagraph = mergeToCurrentParagraph ? paragraph : splitParagraph$1(markerPosition, newPara.format);
	var segmentIndex = newParagraph.segments.indexOf(marker);
	if ((option === null || option === void 0 ? void 0 : option.mergeFormat) == "none" && mergeToCurrentParagraph) {
		newParagraph.segments.forEach(function(segment$1) {
			segment$1.format = __assign(__assign({}, newParagraph.segmentFormat || {}), segment$1.format);
		});
		delete newParagraph.segmentFormat;
	}
	if (segmentIndex >= 0) for (var i$1 = 0; i$1 < newPara.segments.length; i$1++) {
		var segment = newPara.segments[i$1];
		newParagraph.segments.splice(segmentIndex + i$1, 0, segment);
		if (context) {
			if (segment.segmentType == "Entity") context.newEntities.push(segment);
			if (segment.segmentType == "Image") context.newImages.push(segment);
		}
	}
	if (newPara.decorator) {
		newParagraph.decorator = __assign({}, newPara.decorator);
		if (HeadingTags.indexOf(newParagraph.decorator.tagName) > -1) {
			var sourceKeys = getObjectKeys(newParagraph.decorator.format);
			var segmentDecoratorKeys_1 = getObjectKeys(newParagraph.segmentFormat || {});
			sourceKeys.forEach(function(key) {
				var _a$5;
				if (segmentDecoratorKeys_1.indexOf(key) > -1) (_a$5 = newParagraph.segmentFormat) === null || _a$5 === void 0 || delete _a$5[key];
			});
		}
	}
	if (!mergeToCurrentParagraph) newParagraph.format = newPara.format;
	else newParagraph.format = __assign(__assign({}, newParagraph.format), newPara.format);
}
function mergeTables(markerPosition, newTable, source) {
	var _a$5, _b$1;
	var tableContext = markerPosition.tableContext, marker = markerPosition.marker;
	if (tableContext && source.blocks.length == 1 && source.blocks[0] == newTable) {
		var readonlyTable = tableContext.table, colIndex = tableContext.colIndex, rowIndex = tableContext.rowIndex;
		var table = mutateBlock(readonlyTable);
		for (var i$1 = 0; i$1 < newTable.rows.length; i$1++) for (var j$1 = 0; j$1 < newTable.rows[i$1].cells.length; j$1++) {
			var newCell = newTable.rows[i$1].cells[j$1];
			if (i$1 == 0 && colIndex + j$1 >= table.rows[0].cells.length) for (var k$1 = 0; k$1 < table.rows.length; k$1++) {
				var leftCell = (_a$5 = table.rows[k$1]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[colIndex + j$1 - 1];
				table.rows[k$1].cells[colIndex + j$1] = createTableCell(false, false, leftCell === null || leftCell === void 0 ? void 0 : leftCell.isHeader, leftCell === null || leftCell === void 0 ? void 0 : leftCell.format);
			}
			if (j$1 == 0 && rowIndex + i$1 >= table.rows.length) {
				if (!table.rows[rowIndex + i$1]) table.rows[rowIndex + i$1] = {
					cells: [],
					format: {},
					height: 0
				};
				for (var k$1 = 0; k$1 < table.rows[rowIndex].cells.length; k$1++) {
					var aboveCell = (_b$1 = table.rows[rowIndex + i$1 - 1]) === null || _b$1 === void 0 ? void 0 : _b$1.cells[k$1];
					table.rows[rowIndex + i$1].cells[k$1] = createTableCell(false, false, false, aboveCell === null || aboveCell === void 0 ? void 0 : aboveCell.format);
				}
			}
			var oldCell = table.rows[rowIndex + i$1].cells[colIndex + j$1];
			table.rows[rowIndex + i$1].cells[colIndex + j$1] = newCell;
			if (i$1 == 0 && j$1 == 0) {
				var newMarker = createSelectionMarker(marker.format);
				var newPara = addSegment(newCell, newMarker);
				if (markerPosition.path[0] == oldCell) {
					markerPosition.path[0] = newCell;
					markerPosition.marker = newMarker;
					markerPosition.paragraph = newPara;
				}
			}
		}
		normalizeTable$1(table, markerPosition.marker.format);
		applyTableFormat(table, void 0, true);
	} else insertBlock(markerPosition, newTable);
}
function mergeList(markerPosition, newList) {
	splitParagraph$1(markerPosition, newList.format);
	var path = markerPosition.path, paragraph = markerPosition.paragraph;
	var listItemIndex = getClosestAncestorBlockGroupIndex(path, ["ListItem"], ["TableCell"]);
	var listItem = path[listItemIndex];
	var listParent = path[listItemIndex + 1];
	var blockIndex = listParent.blocks.indexOf(listItem || paragraph);
	if (blockIndex >= 0) mutateBlock(listParent).blocks.splice(blockIndex, 0, newList);
	if (listItem) listItem === null || listItem === void 0 || listItem.levels.forEach(function(level, i$1) {
		newList.levels[i$1] = __assign({}, level);
	});
}
function splitParagraph$1(markerPosition, newParaFormat) {
	var paragraph = markerPosition.paragraph, marker = markerPosition.marker, path = markerPosition.path;
	var segmentIndex = paragraph.segments.indexOf(marker);
	var paraIndex = path[0].blocks.indexOf(paragraph);
	var newParagraph = createParagraph(false, __assign(__assign({}, paragraph.format), newParaFormat), paragraph.segmentFormat);
	if (segmentIndex >= 0) newParagraph.segments = paragraph.segments.splice(segmentIndex);
	if (paraIndex >= 0) mutateBlock(path[0]).blocks.splice(paraIndex + 1, 0, newParagraph);
	var listItemIndex = getClosestAncestorBlockGroupIndex(path, ["ListItem"], ["FormatContainer", "TableCell"]);
	var listItem = path[listItemIndex];
	if (listItem) {
		var listParent = listItemIndex >= 0 ? path[listItemIndex + 1] : null;
		var blockIndex = listParent ? listParent.blocks.indexOf(listItem) : -1;
		if (blockIndex >= 0 && listParent) {
			var newListItem = createListItem(listItem.levels, listItem.formatHolder.format);
			if (paraIndex >= 0) newListItem.blocks = listItem.blocks.splice(paraIndex + 1);
			if (blockIndex >= 0) mutateBlock(listParent).blocks.splice(blockIndex + 1, 0, newListItem);
			path[listItemIndex] = newListItem;
		}
	}
	markerPosition.paragraph = newParagraph;
	return newParagraph;
}
function insertBlock(markerPosition, block) {
	var path = markerPosition.path;
	var newPara = splitParagraph$1(markerPosition, block.blockType !== "Paragraph" ? {} : block.format);
	var blockIndex = path[0].blocks.indexOf(newPara);
	if (blockIndex >= 0) mutateBlock(path[0]).blocks.splice(blockIndex, 0, block);
}
function applyDefaultFormat$1(group, format, applyDefaultFormatOption) {
	group.blocks.forEach(function(block) {
		var _a$5;
		mergeBlockFormat(applyDefaultFormatOption, block);
		switch (block.blockType) {
			case "BlockGroup":
				if (block.blockGroupType == "ListItem") mutateBlock(block).formatHolder.format = mergeSegmentFormat(applyDefaultFormatOption, format, block.formatHolder.format);
				applyDefaultFormat$1(block, format, applyDefaultFormatOption);
				break;
			case "Table":
				block.rows.forEach(function(row) {
					return row.cells.forEach(function(cell) {
						applyDefaultFormat$1(cell, format, applyDefaultFormatOption);
					});
				});
				break;
			case "Paragraph":
				var paragraphFormat_1 = ((_a$5 = block.decorator) === null || _a$5 === void 0 ? void 0 : _a$5.format) || {};
				var paragraph = mutateBlock(block);
				paragraph.segments.forEach(function(segment) {
					if (segment.segmentType == "General") applyDefaultFormat$1(segment, format, applyDefaultFormatOption);
					segment.format = mergeSegmentFormat(applyDefaultFormatOption, format, __assign(__assign({}, paragraphFormat_1), segment.format));
					if (segment.link) segment.link.format = mergeLinkFormat(applyDefaultFormatOption, format, segment.link.format);
				});
				if (applyDefaultFormatOption === "keepSourceEmphasisFormat") delete paragraph.decorator;
				break;
		}
	});
}
function mergeBlockFormat(applyDefaultFormatOption, block) {
	if (applyDefaultFormatOption == "keepSourceEmphasisFormat" && block.format.backgroundColor) delete mutateBlock(block).format.backgroundColor;
}
function getSegmentFormatInLinkFormat(targetFormat) {
	var result = {};
	if (targetFormat.backgroundColor) result.backgroundColor = targetFormat.backgroundColor;
	if (targetFormat.underline) result.underline = targetFormat.underline;
	return result;
}
function mergeLinkFormat(applyDefaultFormatOption, targetFormat, sourceFormat) {
	switch (applyDefaultFormatOption) {
		case "mergeAll":
		case "preferSource": return __assign(__assign({}, getSegmentFormatInLinkFormat(targetFormat)), sourceFormat);
		case "keepSourceEmphasisFormat": return __assign(__assign(__assign(__assign({}, getFormatWithoutSegmentFormat(sourceFormat)), getSegmentFormatInLinkFormat(targetFormat)), getSemanticFormat(sourceFormat)), getHyperlinkTextColor(sourceFormat));
		case "preferTarget": return __assign(__assign({}, sourceFormat), getSegmentFormatInLinkFormat(targetFormat));
	}
}
function mergeSegmentFormat(applyDefaultFormatOption, targetFormat, sourceFormat) {
	switch (applyDefaultFormatOption) {
		case "mergeAll":
		case "preferSource": return __assign(__assign({}, targetFormat), sourceFormat);
		case "preferTarget": return __assign(__assign({}, sourceFormat), targetFormat);
		case "keepSourceEmphasisFormat": return __assign(__assign(__assign({}, getFormatWithoutSegmentFormat(sourceFormat)), targetFormat), getSemanticFormat(sourceFormat));
	}
}
function getSemanticFormat(segmentFormat) {
	var result = {};
	var fontWeight = segmentFormat.fontWeight, italic = segmentFormat.italic, underline = segmentFormat.underline;
	if (fontWeight && fontWeight != "normal") result.fontWeight = fontWeight;
	if (italic) result.italic = italic;
	if (underline) result.underline = underline;
	return result;
}
function getFormatWithoutSegmentFormat(sourceFormat) {
	var resultFormat = __assign({}, sourceFormat);
	KeysOfSegmentFormat.forEach(function(key) {
		return delete resultFormat[key];
	});
	return resultFormat;
}
function getHyperlinkTextColor(sourceFormat) {
	var result = {};
	if (sourceFormat.textColor) result.textColor = sourceFormat.textColor;
	return result;
}
var NumberDefinition = createNumberDefinition(true);
var BooleanDefinition = createBooleanDefinition(true);
var ImageMetadataFormatDefinition = createObjectDefinition({
	widthPx: NumberDefinition,
	heightPx: NumberDefinition,
	leftPercent: NumberDefinition,
	rightPercent: NumberDefinition,
	topPercent: NumberDefinition,
	bottomPercent: NumberDefinition,
	angleRad: NumberDefinition,
	src: createStringDefinition(),
	naturalHeight: NumberDefinition,
	naturalWidth: NumberDefinition,
	flippedHorizontal: BooleanDefinition,
	flippedVertical: BooleanDefinition
});
function getImageMetadata(image) {
	return getMetadata(image, ImageMetadataFormatDefinition);
}
function updateImageMetadata(image, callback) {
	return updateMetadata(image, callback, ImageMetadataFormatDefinition);
}
function retrieveModelFormatState(model, pendingFormat, formatState, conflictSolution, domHelper, isInDarkMode, colorHandler) {
	if (conflictSolution === void 0) conflictSolution = "remove";
	var firstTableContext;
	var firstBlock;
	var isFirst = true;
	var isFirstImage = true;
	var isFirstSegment = true;
	var containerFormat = void 0;
	var modelFormat = __assign({}, model.format);
	delete modelFormat.italic;
	delete modelFormat.underline;
	delete modelFormat.fontWeight;
	iterateSelections(model, function(path, tableContext, block, segments) {
		retrieveStructureFormat(formatState, path, isFirst, conflictSolution);
		if (block) if (firstBlock) formatState.isMultilineSelection = true;
		else firstBlock = block;
		if ((block === null || block === void 0 ? void 0 : block.blockType) == "Paragraph") {
			retrieveParagraphFormat(formatState, block, isFirst, conflictSolution);
			segments === null || segments === void 0 || segments.forEach(function(segment) {
				var _a$5, _b$1, _c$1, _d$1;
				if (isFirstSegment || segment.segmentType != "SelectionMarker") {
					var currentFormat = Object.assign({}, block.format, (_a$5 = block.decorator) === null || _a$5 === void 0 ? void 0 : _a$5.format, segment.format, (_b$1 = segment.code) === null || _b$1 === void 0 ? void 0 : _b$1.format, (_c$1 = segment.link) === null || _c$1 === void 0 ? void 0 : _c$1.format, pendingFormat);
					if (!hasAllRequiredFormat(currentFormat)) {
						if (!containerFormat) containerFormat = (_d$1 = domHelper === null || domHelper === void 0 ? void 0 : domHelper.getContainerFormat(isInDarkMode, colorHandler)) !== null && _d$1 !== void 0 ? _d$1 : modelFormat;
						currentFormat = Object.assign({}, containerFormat, currentFormat);
					}
					retrieveSegmentFormat(formatState, isFirst, currentFormat, conflictSolution);
					mergeValue(formatState, "isCodeInline", !!(segment === null || segment === void 0 ? void 0 : segment.code), isFirst, conflictSolution);
				}
				isFirstSegment = false;
				formatState.canUnlink = formatState.canUnlink || !!segment.link;
				formatState.canAddImageAltText = formatState.canAddImageAltText || segments.some(function(segment$1) {
					return segment$1.segmentType == "Image";
				});
				isFirst = false;
				if (segment.segmentType === "Image") if (isFirstImage) {
					retrieveImageFormat(segment, formatState);
					isFirstImage = false;
				} else {
					formatState.imageFormat = void 0;
					formatState.imageEditingMetadata = void 0;
				}
			});
			isFirst = false;
		}
		if (tableContext) if (firstTableContext) {
			var table = firstTableContext.table, colIndex = firstTableContext.colIndex, rowIndex = firstTableContext.rowIndex;
			if (tableContext.table == table && (tableContext.colIndex != colIndex || tableContext.rowIndex != rowIndex)) {
				formatState.canMergeTableCell = true;
				formatState.isMultilineSelection = true;
			}
		} else {
			retrieveTableFormat(tableContext, formatState);
			firstTableContext = tableContext;
		}
	}, { includeListFormatHolder: "never" });
	if (formatState.fontSize) formatState.fontSize = px2Pt(formatState.fontSize);
}
function retrieveSegmentFormat(result, isFirst, mergedFormat, conflictSolution) {
	var _a$5, _b$1;
	if (conflictSolution === void 0) conflictSolution = "remove";
	var superOrSubscript = (_b$1 = (_a$5 = mergedFormat.superOrSubScriptSequence) === null || _a$5 === void 0 ? void 0 : _a$5.split(" ")) === null || _b$1 === void 0 ? void 0 : _b$1.pop();
	mergeValue(result, "isBold", isBold(mergedFormat.fontWeight), isFirst, conflictSolution);
	mergeValue(result, "isItalic", mergedFormat.italic, isFirst, conflictSolution);
	mergeValue(result, "isUnderline", mergedFormat.underline, isFirst, conflictSolution);
	mergeValue(result, "isStrikeThrough", mergedFormat.strikethrough, isFirst, conflictSolution);
	mergeValue(result, "isSuperscript", superOrSubscript == "super", isFirst, conflictSolution);
	mergeValue(result, "isSubscript", superOrSubscript == "sub", isFirst, conflictSolution);
	mergeValue(result, "letterSpacing", mergedFormat.letterSpacing, isFirst, conflictSolution);
	mergeValue(result, "fontName", mergedFormat.fontFamily, isFirst, conflictSolution);
	mergeValue(result, "fontSize", mergedFormat.fontSize, isFirst, conflictSolution, function(val) {
		return parseValueWithUnit(val, void 0, "pt") + "pt";
	});
	mergeValue(result, "backgroundColor", mergedFormat.backgroundColor, isFirst, conflictSolution);
	mergeValue(result, "textColor", mergedFormat.textColor, isFirst, conflictSolution);
	mergeValue(result, "fontWeight", mergedFormat.fontWeight, isFirst, conflictSolution);
	mergeValue(result, "lineHeight", mergedFormat.lineHeight, isFirst, conflictSolution);
}
function retrieveParagraphFormat(result, paragraph, isFirst, conflictSolution) {
	var _a$5;
	if (conflictSolution === void 0) conflictSolution = "remove";
	var headingLevel = parseInt((((_a$5 = paragraph.decorator) === null || _a$5 === void 0 ? void 0 : _a$5.tagName) || "").substring(1));
	var validHeadingLevel = headingLevel >= 1 && headingLevel <= 6 ? headingLevel : void 0;
	mergeValue(result, "marginBottom", paragraph.format.marginBottom, isFirst, conflictSolution);
	mergeValue(result, "marginTop", paragraph.format.marginTop, isFirst, conflictSolution);
	mergeValue(result, "headingLevel", validHeadingLevel, isFirst, conflictSolution);
	mergeValue(result, "textAlign", paragraph.format.textAlign, isFirst, conflictSolution);
	mergeValue(result, "direction", paragraph.format.direction, isFirst, conflictSolution);
}
function retrieveStructureFormat(result, path, isFirst, conflictSolution) {
	var _a$5, _b$1;
	if (conflictSolution === void 0) conflictSolution = "remove";
	var listItemIndex = getClosestAncestorBlockGroupIndex(path, ["ListItem"], []);
	var containerIndex = getClosestAncestorBlockGroupIndex(path, ["FormatContainer"], []);
	if (listItemIndex >= 0) {
		var listItem = path[listItemIndex];
		var listType = (_a$5 = listItem === null || listItem === void 0 ? void 0 : listItem.levels[listItem.levels.length - 1]) === null || _a$5 === void 0 ? void 0 : _a$5.listType;
		mergeValue(result, "isBullet", listType == "UL", isFirst, conflictSolution);
		mergeValue(result, "isNumbering", listType == "OL", isFirst, conflictSolution);
	}
	mergeValue(result, "isBlockQuote", containerIndex >= 0 && ((_b$1 = path[containerIndex]) === null || _b$1 === void 0 ? void 0 : _b$1.tagName) == "blockquote", isFirst, conflictSolution);
}
function retrieveTableFormat(tableContext, result) {
	var tableFormat = getTableMetadata(tableContext.table);
	result.isInTable = true;
	result.tableHasHeader = tableContext.table.rows.some(function(row) {
		return row.cells.some(function(cell) {
			return cell.isHeader;
		});
	});
	if (tableFormat) result.tableFormat = tableFormat;
}
function retrieveImageFormat(image, result) {
	var format = image.format;
	var extractedBorder = extractBorderValues(format["borderTop"]);
	result.imageFormat = {
		borderColor: extractedBorder.color,
		borderWidth: extractedBorder.width,
		borderStyle: extractedBorder.style,
		boxShadow: format.boxShadow,
		borderRadius: format.borderRadius
	};
	result.imageEditingMetadata = getImageMetadata(image);
}
function mergeValue(format, key, newValue, isFirst, conflictSolution, parseFn) {
	if (conflictSolution === void 0) conflictSolution = "remove";
	if (parseFn === void 0) parseFn = function(val) {
		return val;
	};
	if (isFirst) {
		if (newValue !== void 0) format[key] = newValue;
	} else if (parseFn(newValue) !== parseFn(format[key])) switch (conflictSolution) {
		case "remove":
			delete format[key];
			break;
		case "keepFirst": break;
		case "returnMultiple":
			if (typeof format[key] === "string") format[key] = "Multiple";
			else delete format[key];
			break;
	}
}
function px2Pt(px) {
	if (px) {
		var index$1 = px.indexOf("px");
		if (index$1 !== -1 && index$1 === px.length - 2) return Math.round(parseFloat(px) * 75 + .05) / 100 + "pt";
	}
	return px;
}
function hasAllRequiredFormat(format) {
	return !!format.fontFamily && !!format.fontSize && !!format.textColor;
}
var _a$2;
var OrderedListStyleMap$2 = (_a$2 = {}, _a$2[NumberingListType.Decimal] = "decimal", _a$2[NumberingListType.DecimalDash] = "\"${Number}- \"", _a$2[NumberingListType.DecimalParenthesis] = "\"${Number}) \"", _a$2[NumberingListType.DecimalDoubleParenthesis] = "\"(${Number}) \"", _a$2[NumberingListType.LowerAlpha] = "lower-alpha", _a$2[NumberingListType.LowerAlphaDash] = "\"${LowerAlpha}- \"", _a$2[NumberingListType.LowerAlphaParenthesis] = "\"${LowerAlpha}) \"", _a$2[NumberingListType.LowerAlphaDoubleParenthesis] = "\"(${LowerAlpha}) \"", _a$2[NumberingListType.UpperAlpha] = "upper-alpha", _a$2[NumberingListType.UpperAlphaDash] = "\"${UpperAlpha}- \"", _a$2[NumberingListType.UpperAlphaParenthesis] = "\"${UpperAlpha}) \"", _a$2[NumberingListType.UpperAlphaDoubleParenthesis] = "\"(${UpperAlpha}) \"", _a$2[NumberingListType.LowerRoman] = "lower-roman", _a$2[NumberingListType.LowerRomanDash] = "\"${LowerRoman}- \"", _a$2[NumberingListType.LowerRomanParenthesis] = "\"${LowerRoman}) \"", _a$2[NumberingListType.LowerRomanDoubleParenthesis] = "\"(${LowerRoman}) \"", _a$2[NumberingListType.UpperRoman] = "upper-roman", _a$2[NumberingListType.UpperRomanDash] = "\"${UpperRoman}- \"", _a$2[NumberingListType.UpperRomanParenthesis] = "\"${UpperRoman}) \"", _a$2[NumberingListType.UpperRomanDoubleParenthesis] = "\"(${UpperRoman}) \"", _a$2);
var _a$1;
var UnorderedListStyleMap$1 = (_a$1 = {}, _a$1[BulletListType.Disc] = "disc", _a$1[BulletListType.Square] = "square", _a$1[BulletListType.Circle] = "circle", _a$1[BulletListType.Dash] = "\"- \"", _a$1[BulletListType.LongArrow] = "\"➔ \"", _a$1[BulletListType.DoubleLongArrow] = "\"➔ \"", _a$1[BulletListType.ShortArrow] = "\"➢ \"", _a$1[BulletListType.UnfilledArrow] = "\"➪ \"", _a$1[BulletListType.Hyphen] = "\"— \"", _a$1[BulletListType.CheckMark] = "\"✔ \"", _a$1[BulletListType.Xrhombus] = "\"❖ \"", _a$1[BulletListType.BoxShadow] = "\"❑ \"", _a$1);
function getListStyleTypeFromString(listType, bullet) {
	var map = listType == "OL" ? OrderedListStyleMap$2 : UnorderedListStyleMap$1;
	var result = getObjectKeys(map).find(function(key) {
		return map[key] == bullet;
	});
	if (result) return typeof result == "string" ? parseInt(result) : result;
	return result;
}
var ListMetadataDefinition = createObjectDefinition({
	orderedStyleType: createNumberDefinition(true, void 0, NumberingListType.Min, NumberingListType.Max),
	unorderedStyleType: createNumberDefinition(true, void 0, BulletListType.Min, BulletListType.Max),
	applyListStyleFromLevel: createBooleanDefinition(true)
}, true, true);
function getListMetadata(list) {
	return getMetadata(list, ListMetadataDefinition);
}
function updateListMetadata(list, callback) {
	return updateMetadata(list, callback, ListMetadataDefinition);
}
var ChangeSource = {
	AutoLink: "AutoLink",
	CreateLink: "CreateLink",
	Format: "Format",
	ImageResize: "ImageResize",
	Paste: "Paste",
	SetContent: "SetContent",
	Cut: "Cut",
	Drop: "Drop",
	InsertEntity: "InsertEntity",
	SwitchToDarkMode: "SwitchToDarkMode",
	SwitchToLightMode: "SwitchToLightMode",
	ListChain: "ListChain",
	Keyboard: "Keyboard",
	AutoFormat: "AutoFormat",
	Replace: "Replace"
};
function getPath(node, offset, rootNode) {
	var _a$5, _b$1;
	var result = [];
	var parent;
	if (!node || !rootNode.contains(node)) return result;
	if (isNodeOfType(node, "TEXT_NODE")) {
		parent = node.parentNode;
		while (node.previousSibling && isNodeOfType(node.previousSibling, "TEXT_NODE")) {
			offset += ((_a$5 = node.previousSibling.nodeValue) === null || _a$5 === void 0 ? void 0 : _a$5.length) || 0;
			node = node.previousSibling;
		}
		result.unshift(offset);
	} else {
		parent = node;
		node = node.childNodes[offset];
	}
	do {
		offset = 0;
		var isPreviousText = false;
		for (var c$1 = (parent === null || parent === void 0 ? void 0 : parent.firstChild) || null; c$1 && c$1 != node; c$1 = c$1.nextSibling) {
			if (isNodeOfType(c$1, "TEXT_NODE")) {
				if (((_b$1 = c$1.nodeValue) === null || _b$1 === void 0 ? void 0 : _b$1.length) === 0 || isPreviousText) continue;
				isPreviousText = true;
			} else isPreviousText = false;
			offset++;
		}
		result.unshift(offset);
		node = parent;
		parent = (parent === null || parent === void 0 ? void 0 : parent.parentNode) || null;
	} while (node && node != rootNode);
	return result;
}
function createSnapshotSelection(core) {
	var physicalRoot = core.physicalRoot, api = core.api;
	var selection = api.getDOMSelection(core);
	if ((selection === null || selection === void 0 ? void 0 : selection.type) == "range") {
		var _a$5 = selection.range, startContainer = _a$5.startContainer, startOffset = _a$5.startOffset, endContainer = _a$5.endContainer, endOffset = _a$5.endOffset;
		var isDOMChanged = normalizeTableTree(startContainer, physicalRoot);
		if (endContainer != startContainer) isDOMChanged = normalizeTableTree(endContainer, physicalRoot) || isDOMChanged;
		if (isDOMChanged) {
			var newRange = physicalRoot.ownerDocument.createRange();
			newRange.setStart(startContainer, startOffset);
			newRange.setEnd(endContainer, endOffset);
			api.setDOMSelection(core, {
				type: "range",
				range: newRange,
				isReverted: !!selection.isReverted
			}, true);
		}
	}
	switch (selection === null || selection === void 0 ? void 0 : selection.type) {
		case "image": return {
			type: "image",
			imageId: selection.image.id
		};
		case "table": return {
			type: "table",
			tableId: selection.table.id,
			firstColumn: selection.firstColumn,
			lastColumn: selection.lastColumn,
			firstRow: selection.firstRow,
			lastRow: selection.lastRow
		};
		case "range":
			var range = selection.range;
			return {
				type: "range",
				start: getPath(range.startContainer, range.startOffset, physicalRoot),
				end: getPath(range.endContainer, range.endOffset, physicalRoot),
				isReverted: !!selection.isReverted
			};
		default: return {
			type: "range",
			start: [],
			end: [],
			isReverted: false
		};
	}
}
function normalizeTableTree(startNode, root$12) {
	var node = startNode;
	var isDOMChanged = false;
	while (node && root$12.contains(node)) {
		if (isNodeOfType(node, "ELEMENT_NODE") && isElementOfType(node, "table")) isDOMChanged = normalizeTable(node) || isDOMChanged;
		node = node.parentNode;
	}
	return isDOMChanged;
}
function normalizeTable(table) {
	var _a$5;
	var isDOMChanged = false;
	var tbody = null;
	for (var child$1 = table.firstChild; child$1; child$1 = child$1.nextSibling) switch (isNodeOfType(child$1, "ELEMENT_NODE") ? child$1.tagName : null) {
		case "TR":
			if (!tbody) {
				tbody = table.ownerDocument.createElement("tbody");
				table.insertBefore(tbody, child$1);
			}
			tbody.appendChild(child$1);
			child$1 = tbody;
			isDOMChanged = true;
			break;
		case "TBODY":
			if (tbody) {
				moveChildNodes(tbody, child$1, true);
				(_a$5 = child$1.parentNode) === null || _a$5 === void 0 || _a$5.removeChild(child$1);
				child$1 = tbody;
				isDOMChanged = true;
			} else tbody = child$1;
			break;
		default:
			tbody = null;
			break;
	}
	var colgroups = table.querySelectorAll("colgroup");
	var thead = table.querySelector("thead");
	if (thead) colgroups.forEach(function(colgroup) {
		if (!thead.contains(colgroup)) thead.appendChild(colgroup);
	});
	return isDOMChanged;
}
var addUndoSnapshot = function(core, canUndoByBackspace, entityStates) {
	var lifecycle = core.lifecycle, physicalRoot = core.physicalRoot, logicalRoot = core.logicalRoot, undo$1 = core.undo;
	var snapshot$1 = null;
	if (!lifecycle.shadowEditFragment) {
		var beforeAddUndoSnapshotEvent = {
			eventType: "beforeAddUndoSnapshot",
			additionalState: {}
		};
		core.api.triggerEvent(core, beforeAddUndoSnapshotEvent, false);
		var selection = createSnapshotSelection(core);
		var html$1 = physicalRoot.innerHTML;
		if (logicalRoot !== physicalRoot) {
			var entityWrapper = findClosestEntityWrapper(logicalRoot, core.domHelper);
			if (!entityStates && entityWrapper) {
				var entityFormat = parseEntityFormat(entityWrapper);
				if (entityFormat.entityType && entityFormat.id) {
					var event_1 = {
						eventType: "entityOperation",
						operation: "snapshotEntityState",
						entity: {
							type: entityFormat.entityType,
							id: entityFormat.id,
							wrapper: entityWrapper,
							isReadonly: entityFormat.isReadonly
						},
						state: void 0
					};
					core.api.triggerEvent(core, event_1, false);
					if (event_1.state) entityStates = [{
						type: entityFormat.entityType,
						id: entityFormat.id,
						state: event_1.state
					}];
				}
			}
		}
		snapshot$1 = {
			html: html$1,
			additionalState: beforeAddUndoSnapshotEvent.additionalState,
			entityStates,
			isDarkMode: !!lifecycle.isDarkMode,
			selection
		};
		if (logicalRoot !== physicalRoot) snapshot$1.logicalRootPath = getPath(logicalRoot, 0, physicalRoot);
		undo$1.snapshotsManager.addSnapshot(snapshot$1, !!canUndoByBackspace);
		undo$1.snapshotsManager.hasNewContent = false;
	}
	return snapshot$1;
};
function createAriaLiveElement(document$1) {
	var div = document$1.createElement("div");
	div.style.clip = "rect(0px, 0px, 0px, 0px)";
	div.style.clipPath = "inset(100%)";
	div.style.height = "1px";
	div.style.overflow = "hidden";
	div.style.position = "absolute";
	div.style.whiteSpace = "nowrap";
	div.style.width = "1px";
	div.ariaLive = "assertive";
	document$1.body.appendChild(div);
	return div;
}
var DOT_STRING = ".";
var announce = function(core, announceData) {
	var text = announceData.text, defaultStrings = announceData.defaultStrings, _a$5 = announceData.formatStrings, formatStrings = _a$5 === void 0 ? [] : _a$5, _b$1 = announceData.ariaLiveMode, ariaLiveMode = _b$1 === void 0 ? "assertive" : _b$1;
	var announcerStringGetter = core.lifecycle.announcerStringGetter;
	var textToAnnounce = formatString(defaultStrings && (announcerStringGetter === null || announcerStringGetter === void 0 ? void 0 : announcerStringGetter(defaultStrings)) || text, formatStrings);
	if (!core.lifecycle.announceContainer) core.lifecycle.announceContainer = createAriaLiveElement(core.physicalRoot.ownerDocument);
	if (textToAnnounce && core.lifecycle.announceContainer) {
		var announceContainer = core.lifecycle.announceContainer;
		if (announceContainer.ariaLive != ariaLiveMode) announceContainer.ariaLive = ariaLiveMode;
		if (textToAnnounce == announceContainer.textContent) textToAnnounce += DOT_STRING;
		if (announceContainer) announceContainer.textContent = textToAnnounce;
	}
};
function formatString(text, formatStrings) {
	if (text == void 0) return text;
	text = text.replace(/\{(\d+)\}/g, function(_, sub) {
		var replace = formatStrings[parseInt(sub)];
		return replace !== null && replace !== void 0 ? replace : "";
	});
	return text;
}
var attachDomEvent = function(core, eventMap) {
	var disposers = getObjectKeys(eventMap || {}).map(function(key) {
		var _a$5 = eventMap[key], pluginEventType = _a$5.pluginEventType, beforeDispatch = _a$5.beforeDispatch, capture = _a$5.capture;
		var eventName = key;
		var onEvent = function(event$1) {
			if (beforeDispatch) beforeDispatch(event$1);
			if (pluginEventType != null) core.api.triggerEvent(core, {
				eventType: pluginEventType,
				rawEvent: event$1
			}, false);
		};
		core.logicalRoot.addEventListener(eventName, onEvent, { capture });
		return function() {
			core.logicalRoot.removeEventListener(eventName, onEvent, { capture });
		};
	});
	return function() {
		return disposers.forEach(function(disposers$1) {
			return disposers$1();
		});
	};
};
function updateCache(state$1, model, selection) {
	state$1.cachedModel = model;
	if ((selection === null || selection === void 0 ? void 0 : selection.type) == "range") {
		var _a$5 = selection.range, startContainer = _a$5.startContainer, startOffset = _a$5.startOffset, endContainer = _a$5.endContainer, endOffset = _a$5.endOffset;
		state$1.cachedSelection = {
			type: "range",
			isReverted: selection.isReverted,
			start: {
				node: startContainer,
				offset: startOffset
			},
			end: {
				node: endContainer,
				offset: endOffset
			}
		};
	} else state$1.cachedSelection = selection !== null && selection !== void 0 ? selection : void 0;
}
var createContentModel = function(core, option, selectionOverride) {
	var _a$5;
	(_a$5 = core.cache.textMutationObserver) === null || _a$5 === void 0 || _a$5.flushMutations();
	var tryGetFromCache = !option || option.tryGetFromCache && typeof option.recalculateTableSize === "undefined";
	if (!selectionOverride && tryGetFromCache) {
		var cachedModel = core.cache.cachedModel;
		if (cachedModel) return core.lifecycle.shadowEditFragment ? cloneModel(cachedModel, { includeCachedElement: true }) : cachedModel;
	}
	var selection = selectionOverride == "none" ? void 0 : selectionOverride || core.api.getDOMSelection(core) || void 0;
	var saveIndex = !option && !selectionOverride;
	var editorContext = core.api.createEditorContext(core, saveIndex);
	editorContext.recalculateTableSize = option === null || option === void 0 ? void 0 : option.recalculateTableSize;
	var settings = core.environment.domToModelSettings;
	var domToModelContext = option ? createDomToModelContext(editorContext, settings.builtIn, settings.customized, option) : createDomToModelContextWithConfig(settings.calculated, editorContext);
	if (selection) domToModelContext.selection = selection;
	var model = domToContentModel(core.logicalRoot, domToModelContext);
	if (saveIndex) updateCache(core.cache, model, selection);
	return model;
};
var DefaultRootFontSize = 16;
function getRootComputedStyleForContext(document$1) {
	var _a$5;
	var rootComputedStyle = (_a$5 = document$1.defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getComputedStyle(document$1.documentElement);
	return { rootFontSize: parseValueWithUnit(rootComputedStyle === null || rootComputedStyle === void 0 ? void 0 : rootComputedStyle.fontSize) || DefaultRootFontSize };
}
var createEditorContext = function(core, saveIndex) {
	var _a$5, _b$1;
	var lifecycle = core.lifecycle, format = core.format, darkColorHandler = core.darkColorHandler, logicalRoot = core.logicalRoot, cache = core.cache, domHelper = core.domHelper;
	saveIndex = saveIndex && !core.lifecycle.shadowEditFragment;
	var context = __assign({
		isDarkMode: lifecycle.isDarkMode,
		defaultFormat: format.defaultFormat,
		pendingFormat: (_a$5 = format.pendingFormat) !== null && _a$5 !== void 0 ? _a$5 : void 0,
		darkColorHandler,
		addDelimiterForEntity: true,
		allowCacheElement: true,
		domIndexer: saveIndex ? cache.domIndexer : void 0,
		zoomScale: domHelper.calculateZoomScale(),
		experimentalFeatures: (_b$1 = core.experimentalFeatures) !== null && _b$1 !== void 0 ? _b$1 : [],
		paragraphMap: core.cache.paragraphMap
	}, getRootComputedStyleForContext(logicalRoot.ownerDocument));
	if (core.domHelper.isRightToLeft()) context.isRootRtl = true;
	return context;
};
var focus = function(core) {
	var _a$5;
	if (!core.lifecycle.shadowEditFragment) {
		var api = core.api, domHelper = core.domHelper, selection = core.selection;
		if (!domHelper.hasFocus() && ((_a$5 = selection.selection) === null || _a$5 === void 0 ? void 0 : _a$5.type) == "range") api.setDOMSelection(core, selection.selection, true);
		if (!domHelper.hasFocus()) core.logicalRoot.focus();
	}
};
function scrollCaretIntoView(core, selection) {
	var rect = getDOMInsertPointRect(core.physicalRoot.ownerDocument, selection.type == "image" ? {
		node: selection.image,
		offset: 0
	} : selection.isReverted ? {
		node: selection.range.startContainer,
		offset: selection.range.startOffset
	} : {
		node: selection.range.endContainer,
		offset: selection.range.endOffset
	});
	var visibleRect = core.api.getVisibleViewport(core);
	var scrollContainer = core.domEvent.scrollContainer;
	if (rect && visibleRect) scrollRectIntoView(scrollContainer, visibleRect, core.domHelper, rect);
}
var formatContentModel = function(core, formatter, options, domToModelOptions) {
	var _a$5, _b$1;
	var _c$1 = options || {}, onNodeCreated$1 = _c$1.onNodeCreated, rawEvent = _c$1.rawEvent, selectionOverride = _c$1.selectionOverride, scroll = _c$1.scrollCaretIntoView;
	var model = core.api.createContentModel(core, domToModelOptions, selectionOverride);
	var context = {
		newEntities: [],
		deletedEntities: [],
		rawEvent,
		newImages: [],
		paragraphIndexer: core.cache.paragraphMap
	};
	var hasFocus = core.domHelper.hasFocus();
	var changed = formatter(model, context);
	var skipUndoSnapshot = context.skipUndoSnapshot, clearModelCache = context.clearModelCache, entityStates = context.entityStates, canUndoByBackspace = context.canUndoByBackspace;
	if (changed) {
		var isNested = core.undo.isNested;
		var shouldAddSnapshot$1 = (!skipUndoSnapshot || skipUndoSnapshot == "DoNotSkip") && !isNested;
		var shouldMarkNewContent = (skipUndoSnapshot === true || skipUndoSnapshot == "MarkNewContent") && !isNested;
		var selection = void 0;
		if (shouldAddSnapshot$1) {
			core.undo.isNested = true;
			core.api.addUndoSnapshot(core, !!canUndoByBackspace, entityStates);
		}
		try {
			handleImages(core, context);
			selection = (_a$5 = core.api.setContentModel(core, model, hasFocus ? void 0 : { ignoreSelection: true }, onNodeCreated$1)) !== null && _a$5 !== void 0 ? _a$5 : void 0;
			handlePendingFormat(core, context, selection);
			if (scroll && ((selection === null || selection === void 0 ? void 0 : selection.type) == "range" || (selection === null || selection === void 0 ? void 0 : selection.type) == "image")) scrollCaretIntoView(core, selection);
			var eventData = {
				eventType: "contentChanged",
				contentModel: clearModelCache ? void 0 : model,
				selection: clearModelCache ? void 0 : selection,
				source: (options === null || options === void 0 ? void 0 : options.changeSource) || ChangeSource.Format,
				data: (_b$1 = options === null || options === void 0 ? void 0 : options.getChangeData) === null || _b$1 === void 0 ? void 0 : _b$1.call(options),
				formatApiName: options === null || options === void 0 ? void 0 : options.apiName,
				changedEntities: getChangedEntities(context, rawEvent),
				skipUndo: !(shouldMarkNewContent || shouldAddSnapshot$1) || (options === null || options === void 0 ? void 0 : options.changeSource) == ChangeSource.Keyboard
			};
			core.api.triggerEvent(core, eventData, true);
			if (canUndoByBackspace && (selection === null || selection === void 0 ? void 0 : selection.type) == "range") core.undo.autoCompleteInsertPoint = {
				node: selection.range.startContainer,
				offset: selection.range.startOffset
			};
			if (shouldAddSnapshot$1) core.api.addUndoSnapshot(core, false, entityStates);
			if (shouldMarkNewContent) core.undo.snapshotsManager.hasNewContent = true;
		} finally {
			if (!isNested) core.undo.isNested = false;
		}
	} else {
		if (clearModelCache) {
			core.cache.cachedModel = void 0;
			core.cache.cachedSelection = void 0;
		}
		handlePendingFormat(core, context, core.api.getDOMSelection(core));
	}
	if (context.announceData) core.api.announce(core, context.announceData);
};
function handleImages(core, context) {
	if (context.newImages.length > 0) {
		var width = core.domHelper.getClientWidth();
		var maxWidth_1 = Math.max(width, 10);
		context.newImages.forEach(function(image) {
			image.format.maxWidth = maxWidth_1 + "px";
		});
	}
}
function handlePendingFormat(core, context, selection) {
	var _a$5, _b$1;
	var pendingFormat = context.newPendingFormat == "preserve" ? (_a$5 = core.format.pendingFormat) === null || _a$5 === void 0 ? void 0 : _a$5.format : context.newPendingFormat;
	var pendingParagraphFormat = context.newPendingParagraphFormat == "preserve" ? (_b$1 = core.format.pendingFormat) === null || _b$1 === void 0 ? void 0 : _b$1.paragraphFormat : context.newPendingParagraphFormat;
	if ((pendingFormat || pendingParagraphFormat) && (selection === null || selection === void 0 ? void 0 : selection.type) == "range" && selection.range.collapsed) core.format.pendingFormat = {
		format: pendingFormat ? __assign({}, pendingFormat) : void 0,
		paragraphFormat: pendingParagraphFormat ? __assign({}, pendingParagraphFormat) : void 0,
		insertPoint: {
			node: selection.range.startContainer,
			offset: selection.range.startOffset
		}
	};
}
function getChangedEntities(context, rawEvent) {
	return context.autoDetectChangedEntities ? void 0 : context.newEntities.map(function(entity) {
		return {
			entity,
			operation: "newEntity",
			rawEvent
		};
	}).concat(context.deletedEntities.map(function(entry) {
		return {
			entity: entry.entity,
			operation: entry.operation,
			rawEvent
		};
	}));
}
var getDOMSelection = function(core) {
	if (core.lifecycle.shadowEditFragment) return null;
	else {
		var selection = core.selection.selection;
		return selection && (selection.type != "range" || !core.domHelper.hasFocus()) ? selection : getNewSelection(core);
	}
};
function getNewSelection(core) {
	var _a$5;
	var selection = (_a$5 = core.logicalRoot.ownerDocument.defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getSelection();
	var range = selection && selection.rangeCount > 0 ? selection.getRangeAt(0) : null;
	return selection && range && core.logicalRoot.contains(range.commonAncestorContainer) ? {
		type: "range",
		range,
		isReverted: selection.focusNode != range.endContainer || selection.focusOffset != range.endOffset
	} : null;
}
var getVisibleViewport = function(core) {
	var scrollContainer = core.domEvent.scrollContainer;
	return getIntersectedRect$1(scrollContainer == core.physicalRoot ? [scrollContainer] : [scrollContainer, core.physicalRoot]);
};
function getIntersectedRect$1(elements, additionalRects) {
	if (additionalRects === void 0) additionalRects = [];
	var rects = elements.map(function(element) {
		return normalizeRect(element.getBoundingClientRect());
	}).concat(additionalRects).filter(function(rect) {
		return !!rect;
	});
	var result = {
		top: Math.max.apply(Math, __spreadArray([], __read(rects.map(function(r$1) {
			return r$1.top;
		})), false)),
		bottom: Math.min.apply(Math, __spreadArray([], __read(rects.map(function(r$1) {
			return r$1.bottom;
		})), false)),
		left: Math.max.apply(Math, __spreadArray([], __read(rects.map(function(r$1) {
			return r$1.left;
		})), false)),
		right: Math.min.apply(Math, __spreadArray([], __read(rects.map(function(r$1) {
			return r$1.right;
		})), false))
	};
	return result.top < result.bottom && result.left < result.right ? result : null;
}
function restoreSnapshotColors(core, snapshot$1) {
	var isDarkMode = core.lifecycle.isDarkMode;
	core.darkColorHandler.updateKnownColor(isDarkMode);
	if (!!snapshot$1.isDarkMode != !!isDarkMode) transformColor(core.physicalRoot, false, isDarkMode ? "lightToDark" : "darkToLight", core.darkColorHandler);
}
function restoreSnapshotHTML(core, snapshot$1) {
	var _a$5, _b$1;
	var physicalRoot = core.physicalRoot, entityMap = core.entity.entityMap;
	var refNode = physicalRoot.firstChild;
	for (var currentNode = core.domCreator.htmlToDOM(snapshot$1.html).body.firstChild; currentNode;) {
		var next$1 = currentNode.nextSibling;
		var originalEntityElement = tryGetEntityElement(entityMap, currentNode);
		if (originalEntityElement) {
			if (isBlockEntityContainer(originalEntityElement)) {
				for (var node = originalEntityElement.firstChild; node; node = node.nextSibling) if (isNodeOfType(node, "ELEMENT_NODE") && isEntityDelimiter(node)) (_a$5 = core.cache.domIndexer) === null || _a$5 === void 0 || _a$5.clearIndex(node);
			}
			refNode = reuseCachedElement(physicalRoot, originalEntityElement, refNode);
		} else {
			physicalRoot.insertBefore(currentNode, refNode);
			if (isNodeOfType(currentNode, "ELEMENT_NODE")) getAllEntityWrappers(currentNode).forEach(function(element) {
				var _a$6;
				var wrapper = tryGetEntityElement(entityMap, element);
				if (wrapper) {
					if (wrapper == refNode) {
						var markerNode = wrapper.cloneNode();
						physicalRoot.insertBefore(markerNode, refNode);
						refNode = markerNode;
					}
					(_a$6 = element.parentNode) === null || _a$6 === void 0 || _a$6.replaceChild(wrapper, element);
				}
			});
		}
		currentNode = next$1;
	}
	while (refNode) {
		var next$1 = refNode.nextSibling;
		(_b$1 = refNode.parentNode) === null || _b$1 === void 0 || _b$1.removeChild(refNode);
		refNode = next$1;
	}
}
function tryGetEntityElement(entityMap, node) {
	var result = null;
	if (isNodeOfType(node, "ELEMENT_NODE")) {
		if (isEntityElement(node)) result = getEntityWrapperForReuse(entityMap, parseEntityFormat(node).id);
		else if (isBlockEntityContainer(node)) result = tryGetEntityFromContainer(node, entityMap);
	}
	return result;
}
function tryGetEntityFromContainer(element, entityMap) {
	var _a$5;
	for (var node = element.firstChild; node; node = node.nextSibling) if (isEntityElement(node) && isNodeOfType(node, "ELEMENT_NODE")) {
		var parent_1 = (_a$5 = getEntityWrapperForReuse(entityMap, parseEntityFormat(node).id)) === null || _a$5 === void 0 ? void 0 : _a$5.parentElement;
		return isNodeOfType(parent_1, "ELEMENT_NODE") && isBlockEntityContainer(parent_1) ? parent_1 : null;
	}
	return null;
}
function getEntityWrapperForReuse(entityMap, entityId) {
	var entry = entityId ? entityMap[entityId] : void 0;
	return (entry === null || entry === void 0 ? void 0 : entry.canPersist) ? entry.element : null;
}
function getPositionFromPath(node, path) {
	var offset = 0;
	for (var i$1 = 0; i$1 < path.length; i$1++) {
		offset = path[i$1];
		if (i$1 < path.length - 1 && node && isNodeOfType(node, "ELEMENT_NODE") && node.childNodes.length > offset) node = node.childNodes[offset];
		else break;
	}
	return {
		node,
		offset
	};
}
function restoreSnapshotLogicalRoot(core, snapshot$1) {
	if (snapshot$1.logicalRootPath && snapshot$1.logicalRootPath.length > 0) {
		var restoredLogicalRoot = getPositionFromPath(core.physicalRoot, snapshot$1.logicalRootPath).node;
		if (restoredLogicalRoot !== core.logicalRoot) core.api.setLogicalRoot(core, restoredLogicalRoot);
	}
}
function restoreSnapshotSelection(core, snapshot$1) {
	var snapshotSelection = snapshot$1.selection;
	var physicalRoot = core.physicalRoot;
	var domSelection = null;
	try {
		if (snapshotSelection) switch (snapshotSelection.type) {
			case "range":
				var startPos = getPositionFromPath(physicalRoot, snapshotSelection.start);
				var endPos = getPositionFromPath(physicalRoot, snapshotSelection.end);
				var range = physicalRoot.ownerDocument.createRange();
				range.setStart(startPos.node, startPos.offset);
				range.setEnd(endPos.node, endPos.offset);
				domSelection = {
					type: "range",
					range,
					isReverted: snapshotSelection.isReverted
				};
				break;
			case "table":
				var table = physicalRoot.querySelector(getSafeIdSelector(snapshotSelection.tableId));
				if (table) domSelection = {
					type: "table",
					table,
					firstColumn: snapshotSelection.firstColumn,
					firstRow: snapshotSelection.firstRow,
					lastColumn: snapshotSelection.lastColumn,
					lastRow: snapshotSelection.lastRow
				};
				break;
			case "image":
				var image = physicalRoot.querySelector(getSafeIdSelector(snapshotSelection.imageId));
				if (image) domSelection = {
					type: "image",
					image
				};
				break;
		}
		if (domSelection) core.api.setDOMSelection(core, domSelection);
	} catch (_a$5) {}
}
var restoreUndoSnapshot = function(core, snapshot$1) {
	core.api.triggerEvent(core, {
		eventType: "beforeSetContent",
		newContent: snapshot$1.html
	}, true);
	try {
		core.undo.isRestoring = true;
		core.api.setLogicalRoot(core, null);
		restoreSnapshotHTML(core, snapshot$1);
		restoreSnapshotLogicalRoot(core, snapshot$1);
		restoreSnapshotSelection(core, snapshot$1);
		restoreSnapshotColors(core, snapshot$1);
		var event_1 = {
			eventType: "contentChanged",
			additionalState: snapshot$1.additionalState,
			entityStates: snapshot$1.entityStates,
			source: ChangeSource.SetContent
		};
		core.api.triggerEvent(core, event_1, false);
	} finally {
		core.undo.isRestoring = false;
	}
};
var setContentModel = function(core, model, option, onNodeCreated$1, isInitializing) {
	var _a$5, _b$1;
	var editorContext = core.api.createEditorContext(core, true);
	var modelToDomContext = option ? createModelToDomContext(editorContext, core.environment.modelToDomSettings.builtIn, core.environment.modelToDomSettings.customized, option) : createModelToDomContextWithConfig(core.environment.modelToDomSettings.calculated, editorContext);
	modelToDomContext.onNodeCreated = onNodeCreated$1;
	(_a$5 = core.onFixUpModel) === null || _a$5 === void 0 || _a$5.call(core, model);
	var selection = contentModelToDom(core.logicalRoot.ownerDocument, core.logicalRoot, model, modelToDomContext);
	if (!core.lifecycle.shadowEditFragment) {
		(_b$1 = core.cache.textMutationObserver) === null || _b$1 === void 0 || _b$1.flushMutations(true);
		updateCache(core.cache, model, selection);
		if (!(option === null || option === void 0 ? void 0 : option.ignoreSelection) && selection) core.api.setDOMSelection(core, selection);
		else core.selection.selection = selection;
	}
	if (isInitializing) core.lifecycle.rewriteFromModel = modelToDomContext.rewriteFromModel;
	else core.api.triggerEvent(core, __assign({ eventType: "rewriteFromModel" }, modelToDomContext.rewriteFromModel), true);
	return selection;
};
function areSameSelections(sel1, sel2) {
	if (sel1 == sel2) return true;
	switch (sel1.type) {
		case "image": return sel2.type == "image" && sel2.image == sel1.image;
		case "table": return sel2.type == "table" && areSameTableSelections(sel1, sel2);
		case "range":
		default: if (sel2.type == "range") {
			var range1 = sel1.range;
			if (isCacheSelection(sel2)) {
				var start = sel2.start, end = sel2.end;
				return range1.startContainer == start.node && range1.endContainer == end.node && range1.startOffset == start.offset && range1.endOffset == end.offset;
			} else return areSameRanges(range1, sel2.range);
		} else return false;
	}
}
function areSame(o1, o2, keys) {
	return keys.every(function(k$1) {
		return o1[k$1] == o2[k$1];
	});
}
var TableSelectionKeys = [
	"table",
	"firstColumn",
	"lastColumn",
	"firstRow",
	"lastRow"
];
var RangeKeys = [
	"startContainer",
	"endContainer",
	"startOffset",
	"endOffset"
];
function areSameTableSelections(t1, t2) {
	return areSame(t1, t2, TableSelectionKeys);
}
function areSameRanges(r1, r2) {
	return areSame(r1, r2, RangeKeys);
}
function isCacheSelection(sel) {
	return !!sel.start;
}
function addRangeToSelection(doc, range, isReverted) {
	var _a$5;
	if (isReverted === void 0) isReverted = false;
	var selection = (_a$5 = doc.defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getSelection();
	if (selection) {
		var currentRange = selection.rangeCount > 0 && selection.getRangeAt(0);
		if (currentRange && areSameRanges(currentRange, range)) return;
		selection.removeAllRanges();
		if (!isReverted) selection.addRange(range);
		else selection.setBaseAndExtent(range.endContainer, range.endOffset, range.startContainer, range.startOffset);
	}
}
function ensureUniqueId(element, idPrefix$1) {
	idPrefix$1 = element.id || idPrefix$1;
	var doc = element.ownerDocument;
	var i$1 = 0;
	while (!element.id || doc.querySelectorAll(getSafeIdSelector(element.id)).length > 1) element.id = idPrefix$1 + "_" + i$1++;
	return element.id;
}
function findLastedCoInMergedCell(parsedTable, coordinate) {
	var _a$5, _b$1, _c$1, _d$1;
	var row = coordinate.row, col = coordinate.col;
	while (row >= 0 && col >= 0 && row < parsedTable.length && col < ((_b$1 = (_a$5 = parsedTable[row]) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0)) {
		var right = (_c$1 = parsedTable[row]) === null || _c$1 === void 0 ? void 0 : _c$1[col + 1];
		var below = (_d$1 = parsedTable[row + 1]) === null || _d$1 === void 0 ? void 0 : _d$1[col];
		if (right == "spanLeft" || right == "spanBoth") col++;
		else if (below == "spanTop" || below == "spanBoth") row++;
		else return {
			row,
			col
		};
	}
	return null;
}
function findTableCellElement(parsedTable, coordinate) {
	var _a$5, _b$1, _c$1;
	var row = coordinate.row, col = coordinate.col;
	while (row >= 0 && col >= 0 && row < parsedTable.length && col < ((_b$1 = (_a$5 = parsedTable[row]) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0)) {
		var cell = (_c$1 = parsedTable[row]) === null || _c$1 === void 0 ? void 0 : _c$1[col];
		if (!cell) break;
		else if (typeof cell == "object") return {
			cell,
			row,
			col
		};
		else if (cell == "spanLeft" || cell == "spanBoth") col--;
		else row--;
	}
	return null;
}
var CARET_CSS_RULE = "caret-color: transparent";
var HIDE_CURSOR_CSS_KEY = "_DOMSelectionHideCursor";
function toggleCaret(core, isHiding) {
	core.api.setEditorStyle(core, HIDE_CURSOR_CSS_KEY, isHiding ? CARET_CSS_RULE : null);
}
var DOM_SELECTION_CSS_KEY = "_DOMSelection";
var HIDE_SELECTION_CSS_KEY = "_DOMSelectionHideSelection";
var IMAGE_ID = "image";
var TABLE_ID = "table";
var TRANSPARENT_SELECTION_CSS_RULE = "background-color: transparent !important;";
var SELECTION_SELECTOR = "*::selection";
var DEFAULT_SELECTION_BORDER_COLOR$1 = "#DB626C";
var setDOMSelection = function(core, selection, skipSelectionChangedEvent) {
	var _a$5, _b$1, _c$1;
	var existingSelection = core.api.getDOMSelection(core);
	if (existingSelection && selection && areSameSelections(existingSelection, selection)) return;
	var skipReselectOnFocus = core.selection.skipReselectOnFocus;
	var doc = core.physicalRoot.ownerDocument;
	var isDarkMode = core.lifecycle.isDarkMode;
	core.selection.skipReselectOnFocus = true;
	core.api.setEditorStyle(core, DOM_SELECTION_CSS_KEY, null);
	core.api.setEditorStyle(core, HIDE_SELECTION_CSS_KEY, null);
	toggleCaret(core, false);
	try {
		switch (selection === null || selection === void 0 ? void 0 : selection.type) {
			case "image":
				var image = selection.image;
				core.selection.selection = selection;
				var imageSelectionColor = isDarkMode ? core.selection.imageSelectionBorderColorDark : core.selection.imageSelectionBorderColor;
				core.api.setEditorStyle(core, DOM_SELECTION_CSS_KEY, "outline-style:solid!important; outline-color:" + (imageSelectionColor || DEFAULT_SELECTION_BORDER_COLOR$1) + "!important;", [getSafeIdSelector(ensureUniqueId(image, IMAGE_ID))]);
				core.api.setEditorStyle(core, HIDE_SELECTION_CSS_KEY, TRANSPARENT_SELECTION_CSS_RULE, [SELECTION_SELECTOR]);
				setRangeSelection(doc, image, false);
				break;
			case "table":
				var table = selection.table, firstColumn = selection.firstColumn, firstRow = selection.firstRow, lastColumn = selection.lastColumn, lastRow = selection.lastRow;
				var parsedTable = parseTableCells(selection.table);
				var firstCell = {
					row: Math.min(firstRow, lastRow),
					col: Math.min(firstColumn, lastColumn),
					cell: null
				};
				var lastCell = {
					row: Math.max(firstRow, lastRow),
					col: Math.max(firstColumn, lastColumn)
				};
				firstCell = findTableCellElement(parsedTable, firstCell) || firstCell;
				lastCell = findLastedCoInMergedCell(parsedTable, lastCell) || lastCell;
				if (isNaN(firstCell.row) || isNaN(firstCell.col) || isNaN(lastCell.row) || isNaN(lastCell.col)) return;
				selection = {
					type: "table",
					table,
					firstRow: firstCell.row,
					firstColumn: firstCell.col,
					lastRow: lastCell.row,
					lastColumn: lastCell.col,
					tableSelectionInfo: selection.tableSelectionInfo
				};
				var tableSelector = getSafeIdSelector(ensureUniqueId(table, TABLE_ID));
				var tableSelectors = firstCell.row == 0 && firstCell.col == 0 && lastCell.row == parsedTable.length - 1 && lastCell.col == ((_b$1 = (_a$5 = parsedTable[lastCell.row]) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0) - 1 ? [tableSelector, tableSelector + " *"] : handleTableSelected(parsedTable, tableSelector, table, firstCell, lastCell);
				core.selection.selection = selection;
				var tableSelectionColor = isDarkMode ? core.selection.tableCellSelectionBackgroundColorDark : core.selection.tableCellSelectionBackgroundColor;
				core.api.setEditorStyle(core, DOM_SELECTION_CSS_KEY, "background-color:" + tableSelectionColor + "!important;", tableSelectors);
				core.api.setEditorStyle(core, HIDE_SELECTION_CSS_KEY, TRANSPARENT_SELECTION_CSS_RULE, [SELECTION_SELECTOR]);
				toggleCaret(core, true);
				var nodeToSelect = ((_c$1 = firstCell.cell) === null || _c$1 === void 0 ? void 0 : _c$1.firstElementChild) || firstCell.cell;
				if (nodeToSelect) setRangeSelection(doc, nodeToSelect || void 0, true);
				break;
			case "range":
				addRangeToSelection(doc, selection.range, selection.isReverted);
				core.selection.selection = core.domHelper.hasFocus() ? null : selection;
				break;
			default:
				core.selection.selection = null;
				break;
		}
	} finally {
		core.selection.skipReselectOnFocus = skipReselectOnFocus;
	}
	if (!skipSelectionChangedEvent) {
		var eventData = {
			eventType: "selectionChanged",
			newSelection: selection
		};
		core.api.triggerEvent(core, eventData, true);
	}
};
function handleTableSelected(parsedTable, tableSelector, table, firstCell, lastCell) {
	var selectors = [];
	var cont = 0;
	var indexes = toArray(table.childNodes).filter(function(node) {
		return [
			"THEAD",
			"TBODY",
			"TFOOT"
		].indexOf(isNodeOfType(node, "ELEMENT_NODE") ? node.tagName : "") > -1;
	}).map(function(node) {
		var result = {
			el: node.tagName,
			start: cont,
			end: node.childNodes.length + cont
		};
		cont = result.end;
		return result;
	});
	parsedTable.forEach(function(row, rowIndex) {
		var tdCount = 0;
		var midElement = indexes.filter(function(ind) {
			return ind.start <= rowIndex && ind.end > rowIndex;
		})[0];
		var middleElSelector = midElement ? ">" + midElement.el + ">" : ">";
		var currentRow = midElement && rowIndex + 1 >= midElement.start ? rowIndex + 1 - midElement.start : rowIndex + 1;
		for (var cellIndex = 0; cellIndex < row.length; cellIndex++) {
			var cell = row[cellIndex];
			if (typeof cell == "object") {
				tdCount++;
				if (rowIndex >= firstCell.row && rowIndex <= lastCell.row && cellIndex >= firstCell.col && cellIndex <= lastCell.col) {
					var selector = "" + tableSelector + middleElSelector + " tr:nth-child(" + currentRow + ")>" + cell.tagName + ":nth-child(" + tdCount + ")";
					selectors.push(selector, selector + " *");
				}
			}
		}
	});
	return selectors;
}
function setRangeSelection(doc, element, collapse) {
	var _a$5;
	if (element && doc.contains(element)) {
		var range = doc.createRange();
		var isReverted = void 0;
		range.selectNode(element);
		if (collapse) range.collapse();
		else {
			var selection = (_a$5 = doc.defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getSelection();
			var range_1 = selection && selection.rangeCount > 0 && selection.getRangeAt(0);
			if (selection && range_1) isReverted = selection.focusNode != range_1.endContainer || selection.focusOffset != range_1.endOffset;
		}
		addRangeToSelection(doc, range, isReverted);
	}
}
var MAX_RULE_SELECTOR_LENGTH = 9e3;
var CONTENT_DIV_ID = "contentDiv";
var setEditorStyle = function(core, key, cssRule, subSelectors, maxRuleLength) {
	if (maxRuleLength === void 0) maxRuleLength = MAX_RULE_SELECTOR_LENGTH;
	var styleElement = core.lifecycle.styleElements[key];
	if (!styleElement && cssRule) {
		var doc = core.physicalRoot.ownerDocument;
		styleElement = doc.createElement("style");
		doc.head.appendChild(styleElement);
		styleElement.dataset.roosterjsStyleKey = key;
		core.lifecycle.styleElements[key] = styleElement;
	}
	var sheet = styleElement === null || styleElement === void 0 ? void 0 : styleElement.sheet;
	if (sheet) {
		for (var i$1 = sheet.cssRules.length - 1; i$1 >= 0; i$1--) sheet.deleteRule(i$1);
		if (cssRule) {
			var rootSelector = getSafeIdSelector(ensureUniqueId(core.physicalRoot, CONTENT_DIV_ID));
			(!subSelectors ? [rootSelector] : typeof subSelectors === "string" ? [rootSelector + "::" + subSelectors] : buildSelectors(rootSelector, subSelectors, maxRuleLength - cssRule.length - 3)).forEach(function(selector) {
				sheet.insertRule(selector + " {" + cssRule + "}");
			});
		}
	}
};
function buildSelectors(rootSelector, subSelectors, maxLen) {
	var result = [];
	var stringBuilder = [];
	var len = 0;
	subSelectors.forEach(function(subSelector) {
		if (len >= maxLen) {
			result.push(stringBuilder.join(","));
			stringBuilder = [];
			len = 0;
		}
		var selector = rootSelector + " " + subSelector;
		len += selector.length + 1;
		stringBuilder.push(selector);
	});
	result.push(stringBuilder.join(","));
	return result;
}
var setLogicalRoot = function(core, logicalRoot) {
	if (!logicalRoot || core.physicalRoot.contains(logicalRoot)) {
		if (!logicalRoot) logicalRoot = core.physicalRoot;
		if (logicalRoot !== core.logicalRoot) {
			var beforeLogicalRootEvent = {
				eventType: "beforeLogicalRootChange",
				logicalRoot: core.logicalRoot
			};
			core.api.triggerEvent(core, beforeLogicalRootEvent, false);
			core.logicalRoot.contentEditable = "false";
			logicalRoot.contentEditable = "true";
			core.logicalRoot = logicalRoot;
			core.selection.selection = null;
			core.cache.cachedModel = void 0;
			core.cache.cachedSelection = void 0;
			var event_1 = {
				eventType: "logicalRootChanged",
				logicalRoot
			};
			core.api.triggerEvent(core, event_1, false);
		}
	} else return null;
};
var switchShadowEdit = function(editorCore, isOn) {
	var core = editorCore;
	if (isOn != !!core.lifecycle.shadowEditFragment) if (isOn) {
		var model = !core.cache.cachedModel ? core.api.createContentModel(core) : null;
		var fragment = core.logicalRoot.ownerDocument.createDocumentFragment();
		moveChildNodes(fragment, core.logicalRoot.cloneNode(true));
		core.api.triggerEvent(core, { eventType: "enteredShadowEdit" }, false);
		if (!core.cache.cachedModel && model) core.cache.cachedModel = model;
		toggleCaret(core, true);
		core.lifecycle.shadowEditFragment = fragment;
	} else {
		core.lifecycle.shadowEditFragment = null;
		toggleCaret(core, false);
		core.api.triggerEvent(core, { eventType: "leavingShadowEdit" }, false);
		if (core.cache.cachedModel) {
			iterateSelections(core.cache.cachedModel, function() {});
			core.api.setContentModel(core, core.cache.cachedModel, { ignoreSelection: true });
		}
	}
};
var allowedEventsInShadowEdit = [
	"editorReady",
	"beforeDispose",
	"extractContentWithDom",
	"zoomChanged"
];
var triggerEvent = function(core, pluginEvent, broadcast) {
	if ((!core.lifecycle.shadowEditFragment || allowedEventsInShadowEdit.indexOf(pluginEvent.eventType) >= 0) && (broadcast || !core.plugins.some(function(plugin) {
		return handledExclusively(pluginEvent, plugin);
	}))) core.plugins.forEach(function(plugin) {
		if (plugin.onPluginEvent) plugin.onPluginEvent(pluginEvent);
	});
};
function handledExclusively(event$1, plugin) {
	var _a$5;
	if (plugin.onPluginEvent && ((_a$5 = plugin.willHandleEventExclusively) === null || _a$5 === void 0 ? void 0 : _a$5.call(plugin, event$1))) {
		plugin.onPluginEvent(event$1);
		return true;
	}
	return false;
}
var coreApiMap = {
	createContentModel,
	createEditorContext,
	formatContentModel,
	setContentModel,
	setLogicalRoot,
	getDOMSelection,
	setDOMSelection,
	focus,
	addUndoSnapshot,
	restoreUndoSnapshot,
	attachDomEvent,
	triggerEvent,
	switchShadowEdit,
	getVisibleViewport,
	setEditorStyle,
	announce
};
var DarkColorHandlerImpl = function() {
	function DarkColorHandlerImpl$1(root$12, getDarkColor, knownColors, generateColorKey) {
		this.root = root$12;
		this.getDarkColor = getDarkColor;
		this.knownColors = knownColors;
		this.generateColorKey = generateColorKey;
	}
	DarkColorHandlerImpl$1.prototype.updateKnownColor = function(isDarkMode, key, colorPair) {
		var _this = this;
		if (key && colorPair) {
			if (!this.knownColors[key]) this.knownColors[key] = colorPair;
			if (isDarkMode) this.root.style.setProperty(key, colorPair.darkModeColor);
		} else if (isDarkMode) Object.keys(this.knownColors).forEach(function(key$1) {
			_this.root.style.setProperty(key$1, _this.knownColors[key$1].darkModeColor);
		});
	};
	DarkColorHandlerImpl$1.prototype.reset = function() {
		var _this = this;
		Object.keys(this.knownColors).forEach(function(key) {
			_this.root.style.removeProperty(key);
		});
	};
	return DarkColorHandlerImpl$1;
}();
function createDarkColorHandler(root$12, getDarkColor, knownColors, generateColorKey) {
	if (knownColors === void 0) knownColors = {};
	if (generateColorKey === void 0) generateColorKey = defaultGenerateColorKey;
	return new DarkColorHandlerImpl(root$12, getDarkColor, knownColors, generateColorKey);
}
var createTrustedHTMLHandler = function(domCreator) {
	return function(html$1) {
		return domCreator.htmlToDOM(html$1).body.innerHTML;
	};
};
function createDOMCreator(trustedHTMLHandler) {
	return trustedHTMLHandler && isDOMCreator(trustedHTMLHandler) ? trustedHTMLHandler : trustedHTMLHandlerToDOMCreator(trustedHTMLHandler);
}
function isDOMCreator(trustedHTMLHandler) {
	return typeof trustedHTMLHandler.htmlToDOM === "function";
}
var defaultTrustHtmlHandler = function(html$1) {
	return html$1;
};
function trustedHTMLHandlerToDOMCreator(trustedHTMLHandler) {
	var handler = trustedHTMLHandler || defaultTrustHtmlHandler;
	return {
		htmlToDOM: function(html$1) {
			return new DOMParser().parseFromString(handler(html$1), "text/html");
		},
		isBypassed: !trustedHTMLHandler
	};
}
var DOMHelperImpl = function() {
	function DOMHelperImpl$1(contentDiv, options) {
		this.contentDiv = contentDiv;
		this.options = options;
	}
	DOMHelperImpl$1.prototype.queryElements = function(selector) {
		return toArray(this.contentDiv.querySelectorAll(selector));
	};
	DOMHelperImpl$1.prototype.getTextContent = function() {
		return this.contentDiv.textContent || "";
	};
	DOMHelperImpl$1.prototype.isNodeInEditor = function(node, excludeRoot) {
		return excludeRoot && node == this.contentDiv ? false : this.contentDiv.contains(node);
	};
	DOMHelperImpl$1.prototype.calculateZoomScale = function() {
		var _a$5;
		var originalWidth = ((_a$5 = this.contentDiv.getBoundingClientRect()) === null || _a$5 === void 0 ? void 0 : _a$5.width) || 0;
		var visualWidth = this.contentDiv.offsetWidth;
		return visualWidth > 0 && originalWidth > 0 ? Math.round(originalWidth / visualWidth * 100) / 100 : 1;
	};
	DOMHelperImpl$1.prototype.setDomAttribute = function(name, value) {
		if (value === null) this.contentDiv.removeAttribute(name);
		else this.contentDiv.setAttribute(name, value);
	};
	DOMHelperImpl$1.prototype.getDomAttribute = function(name) {
		return this.contentDiv.getAttribute(name);
	};
	DOMHelperImpl$1.prototype.getDomStyle = function(style) {
		return this.contentDiv.style[style];
	};
	DOMHelperImpl$1.prototype.findClosestElementAncestor = function(startFrom, selector) {
		var startElement = isNodeOfType(startFrom, "ELEMENT_NODE") ? startFrom : startFrom.parentElement;
		var closestElement = selector ? startElement === null || startElement === void 0 ? void 0 : startElement.closest(selector) : startElement;
		return closestElement && this.isNodeInEditor(closestElement) && closestElement != this.contentDiv ? closestElement : null;
	};
	DOMHelperImpl$1.prototype.findClosestBlockElement = function(startFrom) {
		var node = startFrom;
		while (node && this.isNodeInEditor(node)) {
			if (isNodeOfType(node, "ELEMENT_NODE") && isBlockElement(node)) return node;
			node = node.parentElement;
		}
		return this.contentDiv;
	};
	DOMHelperImpl$1.prototype.hasFocus = function() {
		var activeElement = this.contentDiv.ownerDocument.activeElement;
		return !!(activeElement && this.contentDiv.contains(activeElement));
	};
	DOMHelperImpl$1.prototype.isRightToLeft = function() {
		var _a$5;
		var contentDiv = this.contentDiv;
		var style = (_a$5 = contentDiv.ownerDocument.defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getComputedStyle(contentDiv);
		return (style === null || style === void 0 ? void 0 : style.direction) == "rtl";
	};
	DOMHelperImpl$1.prototype.getClientWidth = function() {
		var _a$5;
		var contentDiv = this.contentDiv;
		var style = (_a$5 = contentDiv.ownerDocument.defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getComputedStyle(contentDiv);
		var paddingLeft = parseValueWithUnit(style === null || style === void 0 ? void 0 : style.paddingLeft);
		var paddingRight = parseValueWithUnit(style === null || style === void 0 ? void 0 : style.paddingRight);
		return this.contentDiv.clientWidth - (paddingLeft + paddingRight);
	};
	DOMHelperImpl$1.prototype.getClonedRoot = function() {
		if (this.options.cloneIndependentRoot) return this.contentDiv.ownerDocument.implementation.createHTMLDocument().importNode(this.contentDiv, true);
		else return this.contentDiv.cloneNode(true);
	};
	DOMHelperImpl$1.prototype.getContainerFormat = function(isInDarkMode, darkColorHandler) {
		var _a$5, _b$1;
		var window$1 = this.contentDiv.ownerDocument.defaultView;
		var style = window$1 === null || window$1 === void 0 ? void 0 : window$1.getComputedStyle(this.contentDiv);
		return style ? {
			fontSize: style.fontSize,
			fontFamily: style.fontFamily,
			fontWeight: style.fontWeight,
			textColor: getColor(this.contentDiv, false, !!isInDarkMode, darkColorHandler, style.color),
			backgroundColor: getColor(this.contentDiv, true, !!isInDarkMode, darkColorHandler, style.backgroundColor),
			italic: style.fontStyle == "italic",
			letterSpacing: style.letterSpacing,
			lineHeight: style.lineHeight,
			strikethrough: (_a$5 = style.textDecoration) === null || _a$5 === void 0 ? void 0 : _a$5.includes("line-through"),
			superOrSubScriptSequence: style.verticalAlign,
			underline: (_b$1 = style.textDecoration) === null || _b$1 === void 0 ? void 0 : _b$1.includes("underline")
		} : {};
	};
	DOMHelperImpl$1.prototype.getRangesByText = function(text, matchCase, wholeWord) {
		return getRangesByText(this.contentDiv, text, matchCase, wholeWord, true);
	};
	return DOMHelperImpl$1;
}();
function createDOMHelper(contentDiv, options) {
	if (options === void 0) options = {};
	return new DOMHelperImpl(contentDiv, options);
}
var OrderedMapPlaceholderRegex = /\$\{(\w+)\}/;
function getListStyleValue(listType, listStyleType, listNumber) {
	if (listType == "OL") {
		var numberStr = getOrderedListNumberStr(listStyleType, listNumber !== null && listNumber !== void 0 ? listNumber : 1);
		var template = OrderedListStyleMap$2[listStyleType];
		return template ? template.replace(OrderedMapPlaceholderRegex, numberStr) : void 0;
	} else return UnorderedListStyleMap$1[listStyleType];
}
function shouldApplyToItem(listStyleType, listType) {
	var style = listType == "OL" ? OrderedListStyleMap$2[listStyleType] : UnorderedListStyleMap$1[listStyleType];
	return (style === null || style === void 0 ? void 0 : style.indexOf("\"")) >= 0;
}
var listItemMetadataApplier = {
	metadataDefinition: ListMetadataDefinition,
	applierFunction: function(metadata, format, context) {
		var _a$5;
		var depth = context.listFormat.nodeStack.length - 2;
		if (depth >= 0) {
			var listType = (_a$5 = context.listFormat.nodeStack[depth + 1].listType) !== null && _a$5 !== void 0 ? _a$5 : "OL";
			var listStyleType = getAutoListStyleType(listType, metadata !== null && metadata !== void 0 ? metadata : {}, depth);
			if (listStyleType !== void 0) if (shouldApplyToItem(listStyleType, listType)) format.listStyleType = getListStyleValue(listType, listStyleType, context.listFormat.threadItemCounts[depth]);
			else delete format.listStyleType;
		}
	}
};
var listLevelMetadataApplier = {
	metadataDefinition: ListMetadataDefinition,
	applierFunction: function(metadata, format, context) {
		var _a$5;
		var depth = context.listFormat.nodeStack.length - 2;
		if (depth >= 0) {
			var listType = (_a$5 = context.listFormat.nodeStack[depth + 1].listType) !== null && _a$5 !== void 0 ? _a$5 : "OL";
			var listStyleType = getAutoListStyleType(listType, metadata !== null && metadata !== void 0 ? metadata : {}, depth);
			if (listStyleType !== void 0) if (!shouldApplyToItem(listStyleType, listType)) {
				var listStyleTypeFormat = getListStyleValue(listType, listStyleType, context.listFormat.threadItemCounts[depth] > 0 ? context.listFormat.threadItemCounts[depth] : 1);
				if (listStyleTypeFormat) format.listStyleType = listStyleTypeFormat;
			} else delete format.listStyleType;
		}
	}
};
function createDomToModelSettings(options) {
	var _a$5;
	var builtIn = {};
	var customized = (_a$5 = options.defaultDomToModelOptions) !== null && _a$5 !== void 0 ? _a$5 : {};
	return {
		builtIn,
		customized,
		calculated: createDomToModelConfig([builtIn, customized])
	};
}
function createModelToDomSettings(options) {
	var _a$5;
	var builtIn = { metadataAppliers: {
		listItem: listItemMetadataApplier,
		listLevel: listLevelMetadataApplier
	} };
	var customized = (_a$5 = options.defaultModelToDomOptions) !== null && _a$5 !== void 0 ? _a$5 : {};
	return {
		builtIn,
		customized,
		calculated: createModelToDomConfig([builtIn, customized])
	};
}
var idPrefix = "paragraph";
var ParagraphMapImpl = function() {
	function ParagraphMapImpl$1() {
		this.nextId = 0;
		this.paragraphMap = {};
		ParagraphMapImpl$1.prefixNum++;
	}
	ParagraphMapImpl$1.prototype.assignMarkerToModel = function(element, paragraph) {
		var marker = getParagraphMarker(element);
		var paragraphWithMarker = paragraph;
		if (marker) {
			paragraphWithMarker._marker = marker;
			this.paragraphMap[marker] = paragraph;
		} else {
			paragraphWithMarker._marker = this.generateId();
			this.applyMarkerToDom(element, paragraph);
		}
	};
	ParagraphMapImpl$1.prototype.applyMarkerToDom = function(element, paragraph) {
		var paragraphWithMarker = paragraph;
		if (!paragraphWithMarker._marker) paragraphWithMarker._marker = this.generateId();
		var marker = paragraphWithMarker._marker;
		if (marker) {
			setParagraphMarker(element, marker);
			this.paragraphMap[marker] = paragraph;
		}
	};
	ParagraphMapImpl$1.prototype.getParagraphFromMarker = function(markerParagraph) {
		var marker = markerParagraph._marker;
		return marker ? this.paragraphMap[marker] || null : null;
	};
	ParagraphMapImpl$1.prototype.clear = function() {
		this.paragraphMap = {};
	};
	ParagraphMapImpl$1.prototype._reset = function() {
		ParagraphMapImpl$1.prefixNum = 0;
		this.nextId = 0;
	};
	ParagraphMapImpl$1.prototype._getMap = function() {
		return this.paragraphMap;
	};
	ParagraphMapImpl$1.prototype.generateId = function() {
		return idPrefix + "_" + ParagraphMapImpl$1.prefixNum + "_" + this.nextId++;
	};
	ParagraphMapImpl$1.prefixNum = 0;
	return ParagraphMapImpl$1;
}();
function createParagraphMap() {
	return new ParagraphMapImpl();
}
var TextMutationObserverImpl = function() {
	function TextMutationObserverImpl$1(contentDiv, onMutation) {
		var _this = this;
		this.contentDiv = contentDiv;
		this.onMutation = onMutation;
		this.onMutationInternal = function(mutations) {
			var canHandle = true;
			var firstTarget = null;
			var lastTextChangeNode = null;
			var addedNodes = [];
			var removedNodes = [];
			var reconcileText = false;
			var ignoredNodes = /* @__PURE__ */ new Set();
			var includedNodes = /* @__PURE__ */ new Set();
			for (var i$1 = 0; i$1 < mutations.length && canHandle; i$1++) {
				var mutation = mutations[i$1];
				var target = mutation.target;
				if (ignoredNodes.has(target)) continue;
				else if (!includedNodes.has(target)) if (findClosestEntityWrapper(target, _this.domHelper) || findClosestBlockEntityContainer(target, _this.domHelper)) {
					ignoredNodes.add(target);
					continue;
				} else includedNodes.add(target);
				switch (mutation.type) {
					case "attributes":
						if (_this.domHelper.isNodeInEditor(target, true)) if (mutation.attributeName == "id" && isNodeOfType(target, "ELEMENT_NODE")) _this.onMutation({
							type: "elementId",
							element: target
						});
						else canHandle = false;
						break;
					case "characterData":
						if (lastTextChangeNode && lastTextChangeNode != mutation.target) canHandle = false;
						else {
							lastTextChangeNode = mutation.target;
							reconcileText = true;
						}
						break;
					case "childList":
						if (!firstTarget) firstTarget = mutation.target;
						else if (firstTarget != mutation.target) canHandle = false;
						if (canHandle) {
							addedNodes = addedNodes.concat(Array.from(mutation.addedNodes));
							removedNodes = removedNodes.concat(Array.from(mutation.removedNodes));
						}
						break;
				}
			}
			if (canHandle) {
				if (addedNodes.length > 0 || removedNodes.length > 0) _this.onMutation({
					type: "childList",
					addedNodes,
					removedNodes
				});
				if (reconcileText) _this.onMutation({ type: "text" });
			} else _this.onMutation({ type: "unknown" });
		};
		this.observer = new MutationObserver(this.onMutationInternal);
		this.domHelper = createDOMHelper(contentDiv);
	}
	TextMutationObserverImpl$1.prototype.startObserving = function() {
		this.observer.observe(this.contentDiv, {
			subtree: true,
			childList: true,
			attributes: true,
			characterData: true
		});
	};
	TextMutationObserverImpl$1.prototype.stopObserving = function() {
		this.observer.disconnect();
	};
	TextMutationObserverImpl$1.prototype.flushMutations = function(ignoreMutations) {
		var mutations = this.observer.takeRecords();
		if (!ignoreMutations) this.onMutationInternal(mutations);
	};
	return TextMutationObserverImpl$1;
}();
function createTextMutationObserver(contentDiv, onMutation) {
	return new TextMutationObserverImpl(contentDiv, onMutation);
}
function isIndexedSegment(node) {
	var _a$5;
	var _b$1 = (_a$5 = node.__roosterjsContentModel) !== null && _a$5 !== void 0 ? _a$5 : {}, paragraph = _b$1.paragraph, segments = _b$1.segments;
	return paragraph && paragraph.blockType == "Paragraph" && Array.isArray(paragraph.segments) && Array.isArray(segments) && segments.every(function(segment) {
		return paragraph.segments.includes(segment);
	});
}
function isIndexedDelimiter(node) {
	var _a$5;
	var _b$1 = (_a$5 = node.__roosterjsContentModel) !== null && _a$5 !== void 0 ? _a$5 : {}, entity = _b$1.entity, parent = _b$1.parent;
	return (entity === null || entity === void 0 ? void 0 : entity.blockType) == "Entity" && entity.wrapper && (parent === null || parent === void 0 ? void 0 : parent.blockGroupType) && Array.isArray(parent.blocks);
}
function getIndexedSegmentItem(node) {
	return node && isIndexedSegment(node) ? node.__roosterjsContentModel : null;
}
function getIndexedTableItem(element) {
	var index$1 = element.__roosterjsContentModel;
	var table = index$1 === null || index$1 === void 0 ? void 0 : index$1.table;
	if ((table === null || table === void 0 ? void 0 : table.blockType) == "Table" && Array.isArray(table.rows) && table.rows.every(function(x$1) {
		return Array.isArray(x$1 === null || x$1 === void 0 ? void 0 : x$1.cells) && x$1.cells.every(function(y$1) {
			return (y$1 === null || y$1 === void 0 ? void 0 : y$1.blockGroupType) == "TableCell";
		});
	})) return index$1;
	else return null;
}
function unindex(node) {
	delete node.__roosterjsContentModel;
}
var DomIndexerImpl = function() {
	function DomIndexerImpl$1(keepSelectionMarkerWhenEnteringTextNode) {
		this.keepSelectionMarkerWhenEnteringTextNode = keepSelectionMarkerWhenEnteringTextNode;
	}
	DomIndexerImpl$1.prototype.onSegment = function(segmentNode, paragraph, segment) {
		var indexedText = segmentNode;
		indexedText.__roosterjsContentModel = {
			paragraph,
			segments: segment
		};
	};
	DomIndexerImpl$1.prototype.onParagraph = function(paragraphElement) {
		var previousText = null;
		for (var child$1 = paragraphElement.firstChild; child$1; child$1 = child$1.nextSibling) if (isNodeOfType(child$1, "TEXT_NODE")) if (!previousText) previousText = child$1;
		else {
			var item = getIndexedSegmentItem(previousText);
			if (item && isIndexedSegment(child$1)) {
				item.segments = item.segments.concat(child$1.__roosterjsContentModel.segments);
				child$1.__roosterjsContentModel.segments = [];
			}
		}
		else if (isNodeOfType(child$1, "ELEMENT_NODE")) {
			previousText = null;
			this.onParagraph(child$1);
		} else previousText = null;
	};
	DomIndexerImpl$1.prototype.onTable = function(tableElement, table) {
		var indexedTable = tableElement;
		indexedTable.__roosterjsContentModel = { table };
	};
	DomIndexerImpl$1.prototype.onBlockEntity = function(entity, group) {
		this.onBlockEntityDelimiter(entity.wrapper.previousSibling, entity, group);
		this.onBlockEntityDelimiter(entity.wrapper.nextSibling, entity, group);
	};
	DomIndexerImpl$1.prototype.onMergeText = function(targetText, sourceText) {
		var _a$5;
		if (isIndexedSegment(targetText) && isIndexedSegment(sourceText)) {
			if (targetText.nextSibling == sourceText) {
				(_a$5 = targetText.__roosterjsContentModel.segments).push.apply(_a$5, __spreadArray([], __read(sourceText.__roosterjsContentModel.segments), false));
				unindex(sourceText);
			}
		} else {
			unindex(sourceText);
			unindex(targetText);
		}
	};
	DomIndexerImpl$1.prototype.clearIndex = function(container) {
		internalClearIndex(container);
	};
	DomIndexerImpl$1.prototype.reconcileSelection = function(model, newSelection, oldSelection) {
		var _a$5, _b$1;
		var selectionMarker;
		if (oldSelection) {
			var startNode = void 0;
			if (oldSelection.type == "range" && this.isCollapsed(oldSelection) && (startNode = oldSelection.start.node) && isNodeOfType(startNode, "TEXT_NODE") && isIndexedSegment(startNode) && startNode.__roosterjsContentModel.segments.length > 0) this.reconcileTextSelection(startNode);
			else {
				selectionMarker = this.selectionMarkerToKeepWhenEnteringTextNode(oldSelection, newSelection);
				setSelection(model);
			}
		}
		switch (newSelection.type) {
			case "image":
				var indexedImage = getIndexedSegmentItem(newSelection.image);
				var image = indexedImage === null || indexedImage === void 0 ? void 0 : indexedImage.segments[0];
				if (image) {
					image.isSelected = true;
					setSelection(model, image);
					return true;
				} else return false;
			case "table":
				var indexedTable = getIndexedTableItem(newSelection.table);
				if (indexedTable) {
					var firstCell = (_a$5 = indexedTable.table.rows[newSelection.firstRow]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[newSelection.firstColumn];
					var lastCell = (_b$1 = indexedTable.table.rows[newSelection.lastRow]) === null || _b$1 === void 0 ? void 0 : _b$1.cells[newSelection.lastColumn];
					if (firstCell && lastCell) {
						setSelection(model, firstCell, lastCell);
						return true;
					}
				}
				return false;
			case "range":
				var newRange = newSelection.range;
				if (newRange) {
					var startContainer = newRange.startContainer, startOffset = newRange.startOffset, endContainer = newRange.endContainer, endOffset = newRange.endOffset, collapsed = newRange.collapsed;
					delete model.hasRevertedRangeSelection;
					if (collapsed) return !!this.reconcileNodeSelection(startContainer, startOffset, model.format);
					else if (startContainer == endContainer && isNodeOfType(startContainer, "TEXT_NODE")) {
						if (newSelection.isReverted) model.hasRevertedRangeSelection = true;
						return isIndexedSegment(startContainer) && !!this.reconcileTextSelection(startContainer, startOffset, endOffset, selectionMarker);
					} else {
						var marker1 = this.reconcileNodeSelection(startContainer, startOffset);
						var marker2 = this.reconcileNodeSelection(endContainer, endOffset);
						if (marker1 && marker2) {
							if (newSelection.isReverted) model.hasRevertedRangeSelection = true;
							setSelection(model, marker1, marker2);
							return true;
						} else return false;
					}
				}
				break;
		}
		return false;
	};
	DomIndexerImpl$1.prototype.reconcileChildList = function(addedNodes, removedNodes) {
		var canHandle = true;
		var context = { segIndex: -1 };
		var addedNode = addedNodes[0];
		if (addedNodes.length == 1 && isNodeOfType(addedNode, "TEXT_NODE")) canHandle = this.reconcileAddedNode(addedNode, context);
		else if (addedNodes.length > 0) canHandle = false;
		var removedNode = removedNodes[0];
		if (canHandle && removedNodes.length == 1) canHandle = this.reconcileRemovedNode(removedNode, context);
		else if (removedNodes.length > 0) canHandle = false;
		return canHandle && !context.pendingTextNode;
	};
	DomIndexerImpl$1.prototype.reconcileElementId = function(element) {
		var _a$5;
		if (isElementOfType(element, "img")) {
			var indexedImg = getIndexedSegmentItem(element);
			if (((_a$5 = indexedImg === null || indexedImg === void 0 ? void 0 : indexedImg.segments[0]) === null || _a$5 === void 0 ? void 0 : _a$5.segmentType) == "Image") {
				indexedImg.segments[0].format.id = element.id;
				return true;
			} else return false;
		} else if (isElementOfType(element, "table")) {
			var indexedTable = getIndexedTableItem(element);
			if (indexedTable) {
				indexedTable.table.format.id = element.id;
				return true;
			} else return false;
		} else return false;
	};
	DomIndexerImpl$1.prototype.onBlockEntityDelimiter = function(node, entity, parent) {
		if (isNodeOfType(node, "ELEMENT_NODE") && isEntityDelimiter(node) && node.firstChild) {
			var indexedDelimiter = node.firstChild;
			indexedDelimiter.__roosterjsContentModel = {
				entity,
				parent
			};
		}
	};
	DomIndexerImpl$1.prototype.isCollapsed = function(selection) {
		var start = selection.start, end = selection.end;
		return start.node == end.node && start.offset == end.offset;
	};
	DomIndexerImpl$1.prototype.reconcileNodeSelection = function(node, offset, defaultFormat) {
		if (isNodeOfType(node, "TEXT_NODE")) if (isIndexedSegment(node)) return this.reconcileTextSelection(node, offset);
		else if (isIndexedDelimiter(node)) return this.reconcileDelimiterSelection(node, defaultFormat);
		else return;
		else if (offset >= node.childNodes.length) return this.insertMarker(node.lastChild, true);
		else return this.insertMarker(node.childNodes[offset], false);
	};
	DomIndexerImpl$1.prototype.insertMarker = function(node, isAfter) {
		var marker;
		var segmentItem = node && getIndexedSegmentItem(node);
		if (segmentItem) {
			var paragraph = segmentItem.paragraph, segments = segmentItem.segments;
			var index$1 = paragraph.segments.indexOf(segments[0]);
			if (index$1 >= 0) {
				marker = createSelectionMarker((!isAfter && paragraph.segments[index$1 - 1] || paragraph.segments[index$1]).format);
				paragraph.segments.splice(isAfter ? index$1 + 1 : index$1, 0, marker);
			}
		}
		return marker;
	};
	DomIndexerImpl$1.prototype.reconcileTextSelection = function(textNode, startOffset, endOffset, selectionMarker) {
		var _a$5;
		var _b$1, _c$1, _d$1, _e$1, _f$1, _g;
		var _h = textNode.__roosterjsContentModel, paragraph = _h.paragraph, segments = _h.segments;
		var first = segments[0];
		var last = segments[segments.length - 1];
		var selectable;
		if ((first === null || first === void 0 ? void 0 : first.segmentType) == "Text" && (last === null || last === void 0 ? void 0 : last.segmentType) == "Text") {
			var newSegments = [];
			var txt = textNode.nodeValue || "";
			var textSegments = [];
			if (startOffset === void 0) {
				first.text = txt;
				newSegments.push(first);
				textSegments.push(first);
			} else {
				if (startOffset > 0) {
					first.text = txt.substring(0, startOffset);
					newSegments.push(first);
					textSegments.push(first);
				}
				if (endOffset === void 0) {
					var marker = createSelectionMarker((_b$1 = selectionMarker === null || selectionMarker === void 0 ? void 0 : selectionMarker.format) !== null && _b$1 !== void 0 ? _b$1 : first.format);
					newSegments.push(marker);
					if (startOffset < ((_c$1 = textNode.nodeValue) !== null && _c$1 !== void 0 ? _c$1 : "").length) {
						if (first.link) addLink(marker, first.link);
						if (first.code) addCode(marker, first.code);
					}
					selectable = marker;
					endOffset = startOffset;
				} else if (endOffset > startOffset) {
					var middle = createText(txt.substring(startOffset, endOffset), (_d$1 = selectionMarker === null || selectionMarker === void 0 ? void 0 : selectionMarker.format) !== null && _d$1 !== void 0 ? _d$1 : first.format, first.link, first.code);
					middle.isSelected = true;
					newSegments.push(middle);
					textSegments.push(middle);
					selectable = middle;
				}
				if (endOffset < txt.length) {
					var newLast = createText(txt.substring(endOffset), (_e$1 = selectionMarker === null || selectionMarker === void 0 ? void 0 : selectionMarker.format) !== null && _e$1 !== void 0 ? _e$1 : first.format, first.link, first.code);
					newSegments.push(newLast);
					textSegments.push(newLast);
				}
			}
			var firstIndex = paragraph.segments.indexOf(first);
			var lastIndex = paragraph.segments.indexOf(last);
			if (firstIndex >= 0 && lastIndex >= 0) {
				while (firstIndex > 0 && paragraph.segments[firstIndex - 1].segmentType == "SelectionMarker") firstIndex--;
				while (lastIndex < paragraph.segments.length - 1 && paragraph.segments[lastIndex + 1].segmentType == "SelectionMarker") lastIndex++;
				(_a$5 = paragraph.segments).splice.apply(_a$5, __spreadArray([firstIndex, lastIndex - firstIndex + 1], __read(newSegments), false));
			}
			this.onSegment(textNode, paragraph, textSegments);
		} else if ((first === null || first === void 0 ? void 0 : first.segmentType) == "Entity" && first == last) {
			var wrapper = first.wrapper;
			var index$1 = paragraph.segments.indexOf(first);
			var delimiter = textNode.parentElement;
			var isBefore = wrapper.previousSibling == delimiter;
			var isAfter = wrapper.nextSibling == delimiter;
			if (index$1 >= 0 && delimiter && isEntityDelimiter(delimiter) && (isBefore || isAfter)) {
				var marker = createSelectionMarker((_f$1 = selectionMarker === null || selectionMarker === void 0 ? void 0 : selectionMarker.format) !== null && _f$1 !== void 0 ? _f$1 : ((_g = paragraph.segments[isAfter ? index$1 + 1 : index$1 - 1]) !== null && _g !== void 0 ? _g : first).format);
				paragraph.segments.splice(isAfter ? index$1 + 1 : index$1, 0, marker);
				selectable = marker;
			}
		}
		return selectable;
	};
	DomIndexerImpl$1.prototype.reconcileDelimiterSelection = function(node, defaultFormat) {
		var marker;
		var _a$5 = node.__roosterjsContentModel, entity = _a$5.entity, parent = _a$5.parent;
		var index$1 = parent.blocks.indexOf(entity);
		var delimiter = node.parentElement;
		var wrapper = entity.wrapper;
		var isBefore = wrapper.previousSibling == delimiter;
		var isAfter = wrapper.nextSibling == delimiter;
		if (index$1 >= 0 && delimiter && isEntityDelimiter(delimiter) && (isBefore || isAfter)) {
			marker = createSelectionMarker(defaultFormat);
			var para = createParagraph(true, void 0, defaultFormat);
			para.segments.push(marker);
			parent.blocks.splice(isBefore ? index$1 : index$1 + 1, 0, para);
		}
		return marker;
	};
	DomIndexerImpl$1.prototype.reconcileAddedNode = function(node, context) {
		var segmentItem = null;
		var index$1 = -1;
		var existingSegment;
		var previousSibling = node.previousSibling, nextSibling = node.nextSibling;
		if ((segmentItem = getIndexedSegmentItem(getLastLeaf(previousSibling))) && (existingSegment = segmentItem.segments[segmentItem.segments.length - 1]) && (index$1 = segmentItem.paragraph.segments.indexOf(existingSegment)) >= 0) this.indexNode(segmentItem.paragraph, index$1 + 1, node, existingSegment.format);
		else if ((segmentItem = getIndexedSegmentItem(getFirstLeaf(nextSibling))) && (existingSegment = segmentItem.segments[0]) && (index$1 = segmentItem.paragraph.segments.indexOf(existingSegment)) >= 0) this.indexNode(segmentItem.paragraph, index$1, node, existingSegment.format);
		else if (context.paragraph && context.segIndex >= 0) this.indexNode(context.paragraph, context.segIndex, node, context.format);
		else if (context.pendingTextNode === void 0) context.pendingTextNode = node;
		else return false;
		return true;
	};
	DomIndexerImpl$1.prototype.reconcileRemovedNode = function(node, context) {
		var segmentItem = null;
		var removingSegment;
		if (context.segIndex < 0 && !context.paragraph && (segmentItem = getIndexedSegmentItem(node)) && (removingSegment = segmentItem.segments[0])) {
			context.format = removingSegment.format;
			context.paragraph = segmentItem.paragraph;
			context.segIndex = segmentItem.paragraph.segments.indexOf(segmentItem.segments[0]);
			if (context.segIndex < 0) return false;
			for (var i$1 = 0; i$1 < segmentItem.segments.length; i$1++) {
				var index$1 = segmentItem.paragraph.segments.indexOf(segmentItem.segments[i$1]);
				if (index$1 >= 0) segmentItem.paragraph.segments.splice(index$1, 1);
			}
			if (context.pendingTextNode) {
				this.indexNode(context.paragraph, context.segIndex, context.pendingTextNode, segmentItem.segments[0].format);
				context.pendingTextNode = null;
			}
			return true;
		} else return false;
	};
	DomIndexerImpl$1.prototype.indexNode = function(paragraph, index$1, textNode, format) {
		var _a$5;
		var copiedFormat = format ? __assign({}, format) : void 0;
		if (copiedFormat) getObjectKeys(copiedFormat).forEach(function(key) {
			if (EmptySegmentFormat[key] === void 0) delete copiedFormat[key];
		});
		var text = createText((_a$5 = textNode.textContent) !== null && _a$5 !== void 0 ? _a$5 : "", copiedFormat);
		paragraph.segments.splice(index$1, 0, text);
		this.onSegment(textNode, paragraph, [text]);
	};
	DomIndexerImpl$1.prototype.selectionMarkerToKeepWhenEnteringTextNode = function(oldSelection, newSelection) {
		if (this.keepSelectionMarkerWhenEnteringTextNode && oldSelection.type == "range" && this.isCollapsed(oldSelection) && newSelection.type == "range" && isNodeOfType(newSelection.range.commonAncestorContainer, "TEXT_NODE") && newSelection.range.commonAncestorContainer.parentElement == oldSelection.start.node && isIndexedSegment(newSelection.range.commonAncestorContainer) && newSelection.range.commonAncestorContainer.__roosterjsContentModel.paragraph.segments[0].segmentType == "SelectionMarker") return newSelection.range.commonAncestorContainer.__roosterjsContentModel.paragraph.segments[0];
	};
	return DomIndexerImpl$1;
}();
function getLastLeaf(node) {
	while (node === null || node === void 0 ? void 0 : node.lastChild) node = node.lastChild;
	return node;
}
function getFirstLeaf(node) {
	while (node === null || node === void 0 ? void 0 : node.firstChild) node = node.firstChild;
	return node;
}
function internalClearIndex(container) {
	unindex(container);
	for (var node = container.firstChild; node; node = node.nextSibling) internalClearIndex(node);
}
var CachePlugin = function() {
	function CachePlugin$1(option, contentDiv) {
		var _this = this;
		this.editor = null;
		this.onMutation = function(mutation) {
			if (_this.editor) switch (mutation.type) {
				case "childList":
					if (!_this.state.domIndexer.reconcileChildList(mutation.addedNodes, mutation.removedNodes)) _this.invalidateCache();
					break;
				case "text":
					_this.updateCachedModel(_this.editor, true);
					break;
				case "elementId":
					var element = mutation.element;
					if (!_this.state.domIndexer.reconcileElementId(element)) _this.invalidateCache();
					break;
				case "unknown":
					_this.invalidateCache();
					break;
			}
		};
		this.onNativeSelectionChange = function() {
			var _a$5;
			if ((_a$5 = _this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.hasFocus()) _this.updateCachedModel(_this.editor);
		};
		this.state = {
			domIndexer: new DomIndexerImpl(option.experimentalFeatures && option.experimentalFeatures.indexOf("KeepSelectionMarkerWhenEnteringTextNode") >= 0),
			textMutationObserver: createTextMutationObserver(contentDiv, this.onMutation)
		};
		if (option.enableParagraphMap) this.state.paragraphMap = createParagraphMap();
	}
	CachePlugin$1.prototype.getName = function() {
		return "Cache";
	};
	CachePlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		this.editor.getDocument().addEventListener("selectionchange", this.onNativeSelectionChange);
		this.state.textMutationObserver.startObserving();
	};
	CachePlugin$1.prototype.dispose = function() {
		this.state.textMutationObserver.stopObserving();
		if (this.editor) {
			this.editor.getDocument().removeEventListener("selectionchange", this.onNativeSelectionChange);
			this.editor = null;
		}
	};
	CachePlugin$1.prototype.getState = function() {
		return this.state;
	};
	CachePlugin$1.prototype.onPluginEvent = function(event$1) {
		if (!this.editor) return;
		switch (event$1.eventType) {
			case "logicalRootChanged":
				this.invalidateCache();
				this.state.textMutationObserver.stopObserving();
				this.state.textMutationObserver = createTextMutationObserver(event$1.logicalRoot, this.onMutation);
				this.state.textMutationObserver.startObserving();
				break;
			case "selectionChanged":
				this.updateCachedModel(this.editor);
				break;
			case "contentChanged":
				var contentModel = event$1.contentModel, selection = event$1.selection;
				if (contentModel) updateCache(this.state, contentModel, selection);
				else this.invalidateCache();
				break;
		}
	};
	CachePlugin$1.prototype.invalidateCache = function() {
		var _a$5, _b$1;
		if (!((_a$5 = this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.isInShadowEdit())) {
			this.state.cachedModel = void 0;
			this.state.cachedSelection = void 0;
			(_b$1 = this.state.paragraphMap) === null || _b$1 === void 0 || _b$1.clear();
		}
	};
	CachePlugin$1.prototype.updateCachedModel = function(editor, forceUpdate) {
		if (editor.isInShadowEdit()) return;
		var cachedSelection = this.state.cachedSelection;
		this.state.cachedSelection = void 0;
		var newRangeEx = editor.getDOMSelection() || void 0;
		var model = this.state.cachedModel;
		if (forceUpdate || !cachedSelection || !newRangeEx || !areSameSelections(newRangeEx, cachedSelection)) if (!model || !newRangeEx || !this.state.domIndexer.reconcileSelection(model, newRangeEx, cachedSelection)) this.invalidateCache();
		else updateCache(this.state, model, newRangeEx);
		else this.state.cachedSelection = cachedSelection;
	};
	return CachePlugin$1;
}();
function createCachePlugin(option, contentDiv) {
	return new CachePlugin(option, contentDiv);
}
var ContextMenuButton = 2;
var ContextMenuPlugin = function() {
	function ContextMenuPlugin$1(options) {
		var _this = this;
		var _a$5;
		this.editor = null;
		this.disposer = null;
		this.onContextMenuEvent = function(e$1) {
			var _a$6;
			if (_this.editor) {
				var allItems_1 = [];
				var mouseEvent_1 = e$1;
				var pointerEvent = e$1;
				var targetNode_1 = mouseEvent_1.button == ContextMenuButton ? mouseEvent_1.target : (pointerEvent === null || pointerEvent === void 0 ? void 0 : pointerEvent.pointerType) === "touch" || (pointerEvent === null || pointerEvent === void 0 ? void 0 : pointerEvent.pointerType) === "pen" ? pointerEvent.target : _this.getFocusedNode(_this.editor);
				if (targetNode_1) _this.state.contextMenuProviders.forEach(function(provider) {
					var _a$7;
					var items = (_a$7 = provider.getContextMenuItems(targetNode_1, mouseEvent_1)) !== null && _a$7 !== void 0 ? _a$7 : [];
					if ((items === null || items === void 0 ? void 0 : items.length) > 0) {
						if (allItems_1.length > 0) allItems_1.push(null);
						allItems_1.push.apply(allItems_1, __spreadArray([], __read(items), false));
					}
				});
				(_a$6 = _this.editor) === null || _a$6 === void 0 || _a$6.triggerEvent("contextMenu", {
					rawEvent: mouseEvent_1,
					items: allItems_1
				});
			}
		};
		this.state = { contextMenuProviders: ((_a$5 = options.plugins) === null || _a$5 === void 0 ? void 0 : _a$5.filter(isContextMenuProvider)) || [] };
	}
	ContextMenuPlugin$1.prototype.getName = function() {
		return "ContextMenu";
	};
	ContextMenuPlugin$1.prototype.initialize = function(editor) {
		var _this = this;
		this.editor = editor;
		this.disposer = this.editor.attachDomEvent({ contextmenu: { beforeDispatch: function(event$1) {
			return _this.onContextMenuEvent(event$1);
		} } });
	};
	ContextMenuPlugin$1.prototype.dispose = function() {
		var _a$5;
		(_a$5 = this.disposer) === null || _a$5 === void 0 || _a$5.call(this);
		this.disposer = null;
		this.editor = null;
	};
	ContextMenuPlugin$1.prototype.getState = function() {
		return this.state;
	};
	ContextMenuPlugin$1.prototype.getFocusedNode = function(editor) {
		var selection = editor.getDOMSelection();
		if (selection) {
			if (selection.type == "range") selection.range.collapse(true);
			return getSelectionRootNode(selection) || null;
		} else return null;
	};
	return ContextMenuPlugin$1;
}();
function isContextMenuProvider(source) {
	var _a$5;
	return !!((_a$5 = source) === null || _a$5 === void 0 ? void 0 : _a$5.getContextMenuItems);
}
function createContextMenuPlugin(options) {
	return new ContextMenuPlugin(options);
}
function isEmptyBlock(block) {
	if (block && block.blockType == "Paragraph") return block.segments.every(function(segment) {
		return segment.segmentType !== "SelectionMarker" && segment.segmentType == "Br";
	});
	if (block && block.blockType == "BlockGroup") return block.blocks.every(isEmptyBlock);
	return !!block;
}
var deleteEmptyList = function(context) {
	var insertPoint = context.insertPoint;
	if (context.deleteResult == "range" && (insertPoint === null || insertPoint === void 0 ? void 0 : insertPoint.path)) {
		var index$1 = getClosestAncestorBlockGroupIndex(insertPoint.path, ["ListItem"], ["TableCell"]);
		var item = insertPoint.path[index$1];
		if (index$1 >= 0 && item && item.blockGroupType == "ListItem") {
			var listItemIndex = insertPoint.path[index$1 + 1].blocks.indexOf(item);
			var previousBlock = listItemIndex > -1 ? insertPoint.path[index$1 + 1].blocks[listItemIndex - 1] : void 0;
			var nextBlock = listItemIndex > -1 ? insertPoint.path[index$1 + 1].blocks[listItemIndex + 1] : void 0;
			if (hasSelectionInBlockGroup(item) && (!previousBlock || hasSelectionInBlock(previousBlock)) && nextBlock && isEmptyBlock(nextBlock)) mutateBlock(item).levels = [];
		}
	}
};
function adjustImageSelectionOnSafari(editor, selection) {
	if (editor.getEnvironment().isSafari && (selection === null || selection === void 0 ? void 0 : selection.type) == "image") {
		var range = new Range();
		range.setStartBefore(selection.image);
		range.setEndAfter(selection.image);
		editor.setDOMSelection({
			range,
			type: "range",
			isReverted: false
		});
	}
}
function adjustSelectionForCopyCut(pasteModel) {
	var selectionMarker;
	var firstBlock;
	var tableContext;
	iterateSelections(pasteModel, function(_, tableCtxt, block, segments) {
		if (selectionMarker) {
			if (tableCtxt != tableContext && (firstBlock === null || firstBlock === void 0 ? void 0 : firstBlock.segments.includes(selectionMarker))) firstBlock.segments.splice(firstBlock.segments.indexOf(selectionMarker), 1);
			return true;
		}
		var marker = segments === null || segments === void 0 ? void 0 : segments.find(function(segment) {
			return segment.segmentType == "SelectionMarker";
		});
		if (!selectionMarker && marker) {
			tableContext = tableCtxt;
			firstBlock = (block === null || block === void 0 ? void 0 : block.blockType) == "Paragraph" ? block : void 0;
			selectionMarker = marker;
		}
		return false;
	});
}
var BlockEntityClass = "_EBlock";
var OneHundredPercent = "100%";
var InlineBlock = "inline-block";
var onCreateCopyEntityNode = function(model, node) {
	var entityModel = model;
	if (entityModel && entityModel.wrapper && entityModel.blockType == "Entity" && isNodeOfType(node, "ELEMENT_NODE") && isElementOfType(node, "div") && !isBlockElement(entityModel.wrapper) && entityModel.wrapper.style.display == InlineBlock && entityModel.wrapper.style.width == OneHundredPercent) {
		node.classList.add(BlockEntityClass);
		node.style.display = "block";
		node.style.width = "";
	}
};
var pasteBlockEntityParser = function(_, element) {
	if (element.classList.contains(BlockEntityClass)) {
		element.classList.remove(BlockEntityClass);
		element.style.display = InlineBlock;
		element.style.width = OneHundredPercent;
	}
};
function preprocessTable(table) {
	var sel = getSelectedCells(table);
	table.rows = table.rows.map(function(row) {
		return __assign(__assign({}, row), { cells: row.cells.filter(function(cell) {
			return cell.isSelected;
		}) });
	}).filter(function(row) {
		return row.cells.length > 0;
	});
	delete table.format.width;
	table.widths = sel ? table.widths.filter(function(_, index$1) {
		return index$1 >= (sel === null || sel === void 0 ? void 0 : sel.firstColumn) && index$1 <= (sel === null || sel === void 0 ? void 0 : sel.lastColumn);
	}) : [];
}
function pruneUnselectedModel(model) {
	pruneUnselectedModelInternal(model, false);
	unwrap(model);
}
function pruneUnselectedModelInternal(model, isSelectionAfterElement) {
	var e_1, _a$5, _b$1;
	for (var index$1 = model.blocks.length - 1; index$1 >= 0; index$1--) {
		var block = model.blocks[index$1];
		switch (block.blockType) {
			case "BlockGroup":
				pruneUnselectedModelInternal(block, isSelectionAfterElement);
				if (block.blockGroupType == "General" ? block.blocks.length == 0 && !block.isSelected : block.blocks.length == 0) model.blocks.splice(index$1, 1);
				break;
			case "Divider":
			case "Entity":
				if (!block.isSelected) model.blocks.splice(index$1, 1);
				else isSelectionAfterElement = true;
				break;
			case "Paragraph":
				var newSegments = [];
				try {
					for (var _c$1 = (e_1 = void 0, __values(block.segments)), _d$1 = _c$1.next(); !_d$1.done; _d$1 = _c$1.next()) {
						var segment = _d$1.value;
						if (segment.segmentType == "General") {
							pruneUnselectedModel(segment);
							if (segment.blocks.length > 0 || segment.isSelected) newSegments.push(segment);
						} else if (segment.isSelected && segment.segmentType != "SelectionMarker") newSegments.push(segment);
					}
				} catch (e_1_1) {
					e_1 = { error: e_1_1 };
				} finally {
					try {
						if (_d$1 && !_d$1.done && (_a$5 = _c$1.return)) _a$5.call(_c$1);
					} finally {
						if (e_1) throw e_1.error;
					}
				}
				block.segments = newSegments;
				if (block.segments.length == 0) model.blocks.splice(index$1, 1);
				else isSelectionAfterElement = true;
				break;
			case "Table":
				var filteredRows = [];
				for (var i$1 = 0; i$1 < block.rows.length; i$1++) {
					var row = block.rows[i$1];
					for (var j$1 = 0; j$1 < row.cells.length; j$1++) {
						var cell = row.cells[j$1];
						if (!cell.isSelected) pruneUnselectedModelInternal(cell, isSelectionAfterElement);
						else isSelectionAfterElement = true;
					}
					var newCells = [];
					for (var k$1 = 0; k$1 < row.cells.length; k$1++) {
						var cell = row.cells[k$1];
						if (cell.isSelected || cell.blocks.length > 0) newCells.push(cell);
					}
					row.cells = newCells;
					if (row.cells.length > 0) filteredRows.push(row);
				}
				if (!isSelectionAfterElement && filteredRows.length == 1 && filteredRows[0].cells.length == 1 && !filteredRows[0].cells[0].isSelected) {
					var cell = filteredRows[0].cells[0];
					(_b$1 = model.blocks).splice.apply(_b$1, __spreadArray([index$1, 1], __read(cell.blocks), false));
				} else if (filteredRows.length == 0) model.blocks.splice(index$1, 1);
				else block.rows = filteredRows;
				break;
		}
	}
	return isSelectionAfterElement;
}
function unwrap(model) {
	var block = model.blocks[0];
	if (model.blocks.length == 1) {
		while (block.blockType == "BlockGroup") {
			model.blocks = block.blocks;
			block = model.blocks[0];
			if (model.blocks.length > 1) return;
		}
		if (block.blockType == "Paragraph") {
			block.isImplicit = true;
			block.format = {};
			inheritSegmentFormatToChildren(block);
		}
	}
}
function inheritSegmentFormatToChildren(parent) {
	if (parent.segmentFormat !== void 0) for (var index$1 = 0; index$1 < parent.segments.length; index$1++) {
		var segment = parent.segments[index$1];
		segment.format = __assign(__assign({}, parent.segmentFormat), segment.format);
	}
}
var onNodeCreated = function(modelElement, node) {
	if (isNodeOfType(node, "ELEMENT_NODE") && isElementOfType(node, "table")) wrap(node.ownerDocument, node, "div");
	if (isNodeOfType(node, "ELEMENT_NODE") && !node.isContentEditable) node.removeAttribute("contenteditable");
	onCreateCopyEntityNode(modelElement, node);
};
function getContentForCopy(editor, isCut, event$1) {
	var selection = editor.getDOMSelection();
	adjustImageSelectionOnSafari(editor, selection);
	if (selection && (selection.type != "range" || !selection.range.collapsed)) {
		var pasteModel = editor.getContentModelCopy("disconnected");
		pruneUnselectedModel(pasteModel);
		if (selection.type === "table") iterateSelections(pasteModel, function(_, tableContext) {
			if (tableContext === null || tableContext === void 0 ? void 0 : tableContext.table) {
				preprocessTable(tableContext.table);
				return true;
			}
			return false;
		});
		else if (selection.type === "range") adjustSelectionForCopyCut(pasteModel);
		var context = createModelToDomContext();
		context.onNodeCreated = onNodeCreated;
		var doc = editor.getDocument();
		var tempDiv = doc.createElement("div");
		var selectionForCopy = contentModelToDom(doc, tempDiv, pasteModel, context);
		var newRange = selectionForCopy ? domSelectionToRange(doc, selectionForCopy) : null;
		if (newRange) return {
			htmlContent: editor.triggerEvent("beforeCutCopy", {
				clonedRoot: tempDiv,
				range: newRange,
				rawEvent: event$1,
				isCut
			}).clonedRoot,
			textContent: contentModelToText(pasteModel)
		};
	}
	return null;
}
function domSelectionToRange(doc, selection) {
	var _a$5;
	var newRange = null;
	if (selection.type === "table") {
		var table = selection.table;
		var elementToSelect = ((_a$5 = table.parentElement) === null || _a$5 === void 0 ? void 0 : _a$5.childElementCount) == 1 ? table.parentElement : table;
		newRange = doc.createRange();
		newRange.selectNode(elementToSelect);
	} else if (selection.type === "image") {
		newRange = doc.createRange();
		newRange.selectNode(selection.image);
	} else newRange = selection.range;
	return newRange;
}
var HtmlCommentStart = "<!--";
var HtmlCommentStart2 = "<!--";
var HtmlCommentEnd = "-->";
var styleTag = "<style";
var styleClosingTag = "</style>";
var nonWordCharacterRegex$1 = /\W/;
function cleanHtmlComments(html$1) {
	var _a$5;
	var _b$1 = extractHtmlIndexes$1(html$1), styleIndex = _b$1.styleIndex, styleEndIndex = _b$1.styleEndIndex;
	while (styleIndex > -1) {
		html$1 = removeCommentsFromHtml(html$1, HtmlCommentStart, styleEndIndex, styleIndex);
		html$1 = removeCommentsFromHtml(html$1, HtmlCommentStart2, styleEndIndex, styleIndex);
		html$1 = removeCommentsFromHtml(html$1, HtmlCommentEnd, styleEndIndex, styleIndex);
		_a$5 = extractHtmlIndexes$1(html$1, styleEndIndex + 1), styleIndex = _a$5.styleIndex, styleEndIndex = _a$5.styleEndIndex;
	}
	return html$1;
}
function extractHtmlIndexes$1(html$1, startIndex) {
	if (startIndex === void 0) startIndex = 0;
	var htmlLowercase = html$1.toLowerCase();
	var styleIndex = htmlLowercase.indexOf(styleTag, startIndex);
	var currentIndex = styleIndex + styleTag.length;
	var nextChar = html$1.substring(currentIndex, currentIndex + 1);
	while (!nonWordCharacterRegex$1.test(nextChar) && styleIndex > -1) {
		styleIndex = htmlLowercase.indexOf(styleTag, styleIndex + 1);
		currentIndex = styleIndex + styleTag.length;
		nextChar = html$1.substring(currentIndex, currentIndex + 1);
	}
	var styleEndIndex = htmlLowercase.indexOf(styleClosingTag, startIndex);
	return {
		styleIndex,
		styleEndIndex
	};
}
function removeCommentsFromHtml(html$1, marker, endId, startId) {
	var id = html$1.indexOf(marker, startId);
	while (id > -1 && id < endId) {
		html$1 = html$1.substring(0, id) + html$1.substring(id + marker.length);
		id = html$1.indexOf(marker, id + 1);
	}
	return html$1;
}
var containerSizeFormatParser = function(format, element) {
	if (element.tagName == "DIV" || element.tagName == "P") {
		delete format.width;
		delete format.height;
	}
};
var AllowedTags = [
	"a",
	"abbr",
	"address",
	"area",
	"article",
	"aside",
	"b",
	"bdi",
	"bdo",
	"blockquote",
	"body",
	"br",
	"button",
	"canvas",
	"caption",
	"center",
	"cite",
	"code",
	"col",
	"colgroup",
	"data",
	"datalist",
	"dd",
	"del",
	"details",
	"dfn",
	"dialog",
	"dir",
	"div",
	"dl",
	"dt",
	"em",
	"fieldset",
	"figcaption",
	"figure",
	"font",
	"footer",
	"h1",
	"h2",
	"h3",
	"h4",
	"h5",
	"h6",
	"head",
	"header",
	"hgroup",
	"hr",
	"html",
	"i",
	"img",
	"input",
	"ins",
	"kbd",
	"label",
	"legend",
	"li",
	"main",
	"map",
	"mark",
	"menu",
	"menuitem",
	"meter",
	"nav",
	"ol",
	"optgroup",
	"option",
	"output",
	"p",
	"picture",
	"pre",
	"progress",
	"q",
	"rp",
	"rt",
	"ruby",
	"s",
	"samp",
	"section",
	"select",
	"small",
	"span",
	"strike",
	"strong",
	"sub",
	"summary",
	"sup",
	"table",
	"tbody",
	"td",
	"textarea",
	"tfoot",
	"th",
	"thead",
	"time",
	"tr",
	"tt",
	"u",
	"ul",
	"var",
	"wbr",
	"xmp"
];
var DisallowedTags = [
	"applet",
	"audio",
	"base",
	"basefont",
	"embed",
	"frame",
	"frameset",
	"iframe",
	"link",
	"meta",
	"noscript",
	"object",
	"param",
	"script",
	"slot",
	"source",
	"style",
	"template",
	"title",
	"track",
	"video"
];
var VARIABLE_REGEX = /^\s*var\(\s*[a-zA-Z0-9-_]+\s*(,\s*(.*))?\)\s*$/;
var VARIABLE_PREFIX = "var(";
var AllowedAttributes = [
	"accept",
	"align",
	"alt",
	"checked",
	"cite",
	"class",
	"color",
	"cols",
	"colspan",
	"contextmenu",
	"coords",
	"datetime",
	"default",
	"dir",
	"dirname",
	"disabled",
	"download",
	"face",
	"headers",
	"height",
	"hidden",
	"high",
	"href",
	"hreflang",
	"ismap",
	"kind",
	"label",
	"lang",
	"list",
	"low",
	"max",
	"maxlength",
	"media",
	"min",
	"multiple",
	"open",
	"optimum",
	"pattern",
	"placeholder",
	"readonly",
	"rel",
	"required",
	"reversed",
	"rows",
	"rowspan",
	"scope",
	"selected",
	"shape",
	"size",
	"sizes",
	"span",
	"spellcheck",
	"src",
	"srclang",
	"srcset",
	"start",
	"step",
	"style",
	"tabindex",
	"target",
	"title",
	"translate",
	"type",
	"usemap",
	"valign",
	"value",
	"width",
	"wrap",
	"bgColor"
];
var DefaultStyleValue = {
	"background-color": "transparent",
	"border-bottom-color": "rgb(0, 0, 0)",
	"border-bottom-style": "none",
	"border-bottom-width": "0px",
	"border-image-outset": "0",
	"border-image-repeat": "stretch",
	"border-image-slice": "100%",
	"border-image-source": "none",
	"border-image-width": "1",
	"border-left-color": "rgb(0, 0, 0)",
	"border-left-style": "none",
	"border-left-width": "0px",
	"border-right-color": "rgb(0, 0, 0)",
	"border-right-style": "none",
	"border-right-width": "0px",
	"border-top-color": "rgb(0, 0, 0)",
	"border-top-style": "none",
	"border-top-width": "0px",
	"outline-color": "transparent",
	"outline-style": "none",
	"outline-width": "0px",
	overflow: "visible",
	"-webkit-text-stroke-width": "0px",
	"word-wrap": "break-word",
	"margin-left": "0px",
	"margin-right": "0px",
	padding: "0px",
	"padding-top": "0px",
	"padding-left": "0px",
	"padding-right": "0px",
	"padding-bottom": "0px",
	border: "0px",
	"border-top": "0px",
	"border-left": "0px",
	"border-right": "0px",
	"border-bottom": "0px",
	"vertical-align": "baseline",
	float: "none",
	"font-style": "normal",
	"font-variant-ligatures": "normal",
	"font-variant-caps": "normal",
	"font-weight": "400",
	"letter-spacing": "normal",
	orphans: "2",
	"text-align": "start",
	"text-indent": "0px",
	"text-transform": "none",
	widows: "2",
	"word-spacing": "0px",
	"white-space": "normal"
};
function sanitizeElement(element, allowedTags, disallowedTags, styleSanitizers, attributeSanitizers) {
	var tag = element.tagName.toLowerCase();
	var sanitizedElement = disallowedTags.indexOf(tag) >= 0 ? null : createSanitizedElement(element.ownerDocument, allowedTags.indexOf(tag) >= 0 ? tag : "span", element.attributes, styleSanitizers, attributeSanitizers);
	if (sanitizedElement) for (var child$1 = element.firstChild; child$1; child$1 = child$1.nextSibling) {
		var newChild = isNodeOfType(child$1, "ELEMENT_NODE") ? sanitizeElement(child$1, allowedTags, disallowedTags, styleSanitizers, attributeSanitizers) : isNodeOfType(child$1, "TEXT_NODE") ? child$1.cloneNode() : null;
		if (newChild) sanitizedElement === null || sanitizedElement === void 0 || sanitizedElement.appendChild(newChild);
	}
	return sanitizedElement;
}
function createSanitizedElement(doc, tag, attributes, styleSanitizers, attributeSanitizers) {
	var element = doc.createElement(tag);
	for (var i$1 = 0; i$1 < attributes.length; i$1++) {
		var attribute = attributes[i$1];
		var name_1 = attribute.name.toLowerCase().trim();
		var value = attribute.value;
		var sanitizer = attributeSanitizers === null || attributeSanitizers === void 0 ? void 0 : attributeSanitizers[name_1];
		var newValue = name_1 == "style" ? processStyles(tag, value, styleSanitizers) : typeof sanitizer == "function" ? sanitizer(value, tag) : typeof sanitizer === "boolean" ? sanitizer ? value : null : AllowedAttributes.indexOf(name_1) >= 0 || name_1.indexOf("data-") == 0 ? value : null;
		if (newValue !== null && newValue !== void 0 && !newValue.match(/s\n*c\n*r\n*i\n*p\n*t\n*:/i)) element.setAttribute(name_1, newValue);
	}
	return element;
}
function processStyles(tagName, value, styleSanitizers) {
	var pairs = value.split(";");
	var result = [];
	pairs.forEach(function(pair) {
		var valueIndex = pair.indexOf(":");
		var name = pair.slice(0, valueIndex).trim();
		var value$1 = pair.slice(valueIndex + 1).trim();
		if (name && value$1) {
			if (isCssVariable(value$1)) value$1 = processCssVariable(value$1);
			var sanitizer = styleSanitizers === null || styleSanitizers === void 0 ? void 0 : styleSanitizers[name];
			var sanitizedValue = typeof sanitizer == "function" ? sanitizer(value$1, tagName) : sanitizer === false ? null : value$1;
			if (!!sanitizedValue && sanitizedValue != "inherit" && sanitizedValue != "initial" && sanitizedValue.indexOf("expression") < 0 && !name.startsWith("-") && DefaultStyleValue[name] != sanitizedValue) result.push(name + ":" + sanitizedValue);
		}
	});
	return result.join(";");
}
function processCssVariable(value) {
	var match = VARIABLE_REGEX.exec(value);
	return (match === null || match === void 0 ? void 0 : match[2]) || "";
}
function isCssVariable(value) {
	return value.indexOf(VARIABLE_PREFIX) == 0;
}
var DefaultStyleSanitizers$1 = { position: false };
function createPasteEntityProcessor(options) {
	var allowedTags = AllowedTags.concat(options.additionalAllowedTags);
	var disallowedTags = DisallowedTags.concat(options.additionalDisallowedTags);
	var styleSanitizers = Object.assign({}, DefaultStyleSanitizers$1, options.styleSanitizers);
	var attrSanitizers = options.attributeSanitizers;
	return function(group, element, context) {
		var sanitizedElement = sanitizeElement(element, allowedTags, disallowedTags, styleSanitizers, attrSanitizers);
		if (sanitizedElement) context.defaultElementProcessors.entity(group, sanitizedElement, context);
	};
}
var removeDisplayFlex = function(value) {
	return value == "flex" ? null : value;
};
var DefaultStyleSanitizers = {
	position: false,
	display: removeDisplayFlex
};
function createPasteGeneralProcessor(options) {
	var allowedTags = AllowedTags.concat(options.additionalAllowedTags);
	var disallowedTags = DisallowedTags.concat(options.additionalDisallowedTags);
	var styleSanitizers = Object.assign({}, DefaultStyleSanitizers, options.styleSanitizers);
	var attrSanitizers = options.attributeSanitizers;
	return function(group, element, context) {
		var tag = element.tagName.toLowerCase();
		var processor = allowedTags.indexOf(tag) >= 0 ? function(group$1, element$1, context$1) {
			var _a$5, _b$1;
			var sanitizedElement = createSanitizedElement(element$1.ownerDocument, element$1.tagName, element$1.attributes, styleSanitizers, attrSanitizers);
			moveChildNodes(sanitizedElement, element$1);
			(_b$1 = (_a$5 = context$1.defaultElementProcessors)["*"]) === null || _b$1 === void 0 || _b$1.call(_a$5, group$1, sanitizedElement, context$1);
		} : disallowedTags.indexOf(tag) >= 0 ? void 0 : context.defaultElementProcessors.span;
		processor === null || processor === void 0 || processor(group, element, context);
	};
}
var pasteDisplayFormatParser = function(format, element) {
	var display = element.style.display;
	if (display && display != "flex") format.display = display;
};
var pasteTextProcessor = function(group, text, context) {
	var _a$5, _b$1;
	var whiteSpace = context.blockFormat.whiteSpace;
	if (isWhiteSpacePreserved(whiteSpace)) text.nodeValue = (_b$1 = (_a$5 = text.nodeValue) === null || _a$5 === void 0 ? void 0 : _a$5.replace(/\u0020\u0020/g, " \xA0")) !== null && _b$1 !== void 0 ? _b$1 : "";
	context.defaultElementProcessors["#text"](group, text, context);
};
var WhiteSpacePre = "pre";
var pasteWhiteSpaceFormatParser = function(format, element, context, defaultStyle) {
	var _a$5, _b$1;
	if (element.style.whiteSpace != WhiteSpacePre) (_b$1 = (_a$5 = context.defaultFormatParsers).whiteSpace) === null || _b$1 === void 0 || _b$1.call(_a$5, format, element, context, defaultStyle);
};
var DefaultSanitizingOption = {
	processorOverride: {},
	formatParserOverride: {},
	additionalFormatParsers: {},
	additionalAllowedTags: [],
	additionalDisallowedTags: [],
	styleSanitizers: {},
	attributeSanitizers: {},
	processNonVisibleElements: false
};
function createDomToModelContextForSanitizing(document$1, defaultFormat, defaultOption, additionalSanitizingOption) {
	var sanitizingOption = __assign(__assign({}, DefaultSanitizingOption), additionalSanitizingOption);
	return createDomToModelContext(__assign(__assign({ defaultFormat }, getRootComputedStyleForContext(document$1)), { experimentalFeatures: [] }), defaultOption, {
		processorOverride: {
			"#text": pasteTextProcessor,
			entity: createPasteEntityProcessor(sanitizingOption),
			"*": createPasteGeneralProcessor(sanitizingOption)
		},
		formatParserOverride: {
			display: pasteDisplayFormatParser,
			whiteSpace: pasteWhiteSpaceFormatParser
		},
		additionalFormatParsers: {
			container: [containerSizeFormatParser],
			entity: [pasteBlockEntityParser]
		}
	}, sanitizingOption);
}
var BlackColor = "rgb(0,0,0)";
var CloneOption = { includeCachedElement: function(node, type) {
	return type == "cache" ? void 0 : node;
} };
function cloneModelForPaste(model) {
	return cloneModel(model, CloneOption);
}
function mergePasteContent(editor, eventResult, isFirstPaste) {
	var fragment = eventResult.fragment, domToModelOption = eventResult.domToModelOption, customizedMerge = eventResult.customizedMerge, pasteType = eventResult.pasteType, clipboardData = eventResult.clipboardData, containsBlockElements = eventResult.containsBlockElements;
	editor.formatContentModel(function(model, context) {
		if (!isFirstPaste && clipboardData.modelBeforePaste) model.blocks = cloneModelForPaste(clipboardData.modelBeforePaste).blocks;
		var domToModelContext = createDomToModelContextForSanitizing(editor.getDocument(), void 0, editor.getEnvironment().domToModelSettings.customized, domToModelOption);
		domToModelContext.segmentFormat = getSegmentFormatForPaste(model, pasteType);
		var pasteModel = domToContentModel(fragment, domToModelContext);
		var mergeOption = {
			mergeFormat: pasteType == "mergeFormat" ? "keepSourceEmphasisFormat" : "none",
			mergeTable: shouldMergeTable(pasteModel),
			addParagraphAfterMergedContent: containsBlockElements
		};
		var insertPoint = customizedMerge ? customizedMerge(model, pasteModel) : mergeModel(model, pasteModel, context, mergeOption);
		if (insertPoint) context.newPendingFormat = __assign(__assign(__assign({}, EmptySegmentFormat), model.format), pasteType == "normal" && !containsBlockElements ? getLastSegmentFormat(pasteModel) : insertPoint.marker.format);
		return true;
	}, {
		changeSource: ChangeSource.Paste,
		getChangeData: function() {
			return clipboardData;
		},
		scrollCaretIntoView: true,
		apiName: "paste"
	});
}
function getSegmentFormatForPaste(model, pasteType) {
	var selectedSegment = getSelectedSegments(model, true)[0];
	if (selectedSegment) {
		var result = getSegmentTextFormat(selectedSegment);
		if (pasteType == "normal") result.textColor = BlackColor;
		return result;
	}
	return {};
}
function shouldMergeTable(pasteModel) {
	if (pasteModel.blocks.length == 2 && pasteModel.blocks[0].blockType === "Table" && pasteModel.blocks[1].blockType === "Paragraph" && pasteModel.blocks[1].segments.length === 1 && pasteModel.blocks[1].segments[0].segmentType === "Br") pasteModel.blocks.splice(1);
	return pasteModel.blocks.length === 1 && pasteModel.blocks[0].blockType === "Table";
}
function getLastSegmentFormat(pasteModel) {
	if (pasteModel.blocks.length == 1) {
		var firstBlock = __read(pasteModel.blocks, 1)[0];
		if (firstBlock.blockType == "Paragraph") {
			var segment = firstBlock.segments[firstBlock.segments.length - 1];
			return __assign({}, segment.format);
		}
	}
	return {};
}
function splitSelectors(selectorText) {
	return selectorText.split(/(?![^(]*\)),/).map(function(s$1) {
		return s$1.trim();
	});
}
function retrieveCssRules(doc) {
	var styles = toArray(doc.querySelectorAll("style"));
	var result = [];
	styles.forEach(function(styleNode) {
		var _a$5;
		var sheet = styleNode.sheet;
		if (sheet) for (var ruleIndex = 0; ruleIndex < sheet.cssRules.length; ruleIndex++) {
			var rule = sheet.cssRules[ruleIndex];
			if (rule.type == CSSRule.STYLE_RULE && rule.selectorText) result.push({
				selectors: splitSelectors(rule.selectorText),
				text: rule.style.cssText
			});
		}
		(_a$5 = styleNode.parentNode) === null || _a$5 === void 0 || _a$5.removeChild(styleNode);
	});
	return result;
}
function convertInlineCss(root$12, cssRules) {
	var _loop_1 = function(i$2) {
		var e_1, _a$5;
		var _b$1 = cssRules[i$2], selectors = _b$1.selectors, text = _b$1.text;
		try {
			for (var selectors_1 = (e_1 = void 0, __values(selectors)), selectors_1_1 = selectors_1.next(); !selectors_1_1.done; selectors_1_1 = selectors_1.next()) {
				var selector = selectors_1_1.value;
				if (!selector || !selector.trim()) continue;
				toArray(root$12.querySelectorAll(selector)).forEach(function(node) {
					return node.setAttribute("style", text + (node.getAttribute("style") || ""));
				});
			}
		} catch (e_1_1) {
			e_1 = { error: e_1_1 };
		} finally {
			try {
				if (selectors_1_1 && !selectors_1_1.done && (_a$5 = selectors_1.return)) _a$5.call(selectors_1);
			} finally {
				if (e_1) throw e_1.error;
			}
		}
	};
	for (var i$1 = cssRules.length - 1; i$1 >= 0; i$1--) _loop_1(i$1);
}
var NBSP_HTML = "\xA0";
var ENSP_HTML = " ";
var TAB_SPACES = 6;
function createPasteFragment(document$1, clipboardData, pasteType, root$12) {
	if (!clipboardData.text && pasteType === "asPlainText" && root$12) clipboardData.text = root$12.textContent || clipboardData.text;
	var imageDataUri = clipboardData.imageDataUri, text = clipboardData.text;
	var fragment = document$1.createDocumentFragment();
	if (pasteType == "asImage" && imageDataUri || pasteType != "asPlainText" && !text && imageDataUri) {
		var img = document$1.createElement("img");
		img.style.maxWidth = "100%";
		img.src = imageDataUri;
		fragment.appendChild(img);
	} else if (pasteType != "asPlainText" && root$12) moveChildNodes(fragment, root$12);
	else if (text) text.split("\n").forEach(function(line, index$1, lines) {
		line = line.replace(/^ /g, NBSP_HTML).replace(/ $/g, NBSP_HTML).replace(/\r/g, "").replace(/ {2}/g, " " + NBSP_HTML);
		if (line.includes("	")) line = transformTabCharacters(line);
		var textNode = document$1.createTextNode(line);
		if (lines.length == 2 && index$1 == 0) {
			fragment.appendChild(textNode);
			fragment.appendChild(document$1.createElement("br"));
		} else if (index$1 > 0 && index$1 < lines.length - 1) fragment.appendChild(wrap(document$1, line == "" ? document$1.createElement("br") : textNode, "div"));
		else fragment.appendChild(textNode);
	});
	return fragment;
}
function transformTabCharacters(input, initialOffset) {
	if (initialOffset === void 0) initialOffset = 0;
	var line = input;
	var tIndex;
	while ((tIndex = line.indexOf("	")) != -1) {
		var lineBefore = line.slice(0, tIndex);
		var lineAfter = line.slice(tIndex + 1);
		var tabCount = TAB_SPACES - (lineBefore.length + initialOffset) % TAB_SPACES;
		line = lineBefore + Array(tabCount).fill(ENSP_HTML).join("") + lineAfter;
	}
	return line;
}
function generatePasteOptionFromPlugins(editor, clipboardData, fragment, htmlFromClipboard, pasteType) {
	var _a$5, _b$1;
	var domToModelOption = {
		additionalAllowedTags: [],
		additionalDisallowedTags: [],
		additionalFormatParsers: {},
		formatParserOverride: {},
		processorOverride: {},
		styleSanitizers: {},
		attributeSanitizers: {},
		processNonVisibleElements: !!editor.getEnvironment().domToModelSettings.customized.processNonVisibleElements
	};
	var event$1 = {
		eventType: "beforePaste",
		clipboardData,
		fragment,
		htmlBefore: (_a$5 = htmlFromClipboard.htmlBefore) !== null && _a$5 !== void 0 ? _a$5 : "",
		htmlAfter: (_b$1 = htmlFromClipboard.htmlAfter) !== null && _b$1 !== void 0 ? _b$1 : "",
		htmlAttributes: htmlFromClipboard.metadata,
		pasteType,
		domToModelOption,
		containsBlockElements: !!htmlFromClipboard.containsBlockElements
	};
	return editor.triggerEvent("beforePaste", event$1, true);
}
var START_FRAGMENT = "<!--StartFragment-->";
var END_FRAGMENT = "<!--EndFragment-->";
function retrieveHtmlInfo(doc, clipboardData) {
	var result = {
		metadata: {},
		globalCssRules: []
	};
	if (doc) {
		result = __assign(__assign({}, retrieveHtmlStrings(clipboardData)), {
			globalCssRules: retrieveCssRules(doc),
			metadata: retrieveMetadata(doc),
			containsBlockElements: checkBlockElements(doc)
		});
		clipboardData.htmlFirstLevelChildTags = retrieveTopLevelTags(doc);
	}
	return result;
}
function retrieveTopLevelTags(doc) {
	var _a$5;
	var topLevelTags = [];
	for (var child$1 = doc.body.firstChild; child$1; child$1 = child$1.nextSibling) if (isNodeOfType(child$1, "TEXT_NODE")) {
		if ((_a$5 = child$1.nodeValue) === null || _a$5 === void 0 ? void 0 : _a$5.replace(/(\r\n|\r|\n)/gm, "").trim()) topLevelTags.push("");
	} else if (isNodeOfType(child$1, "ELEMENT_NODE")) topLevelTags.push(child$1.tagName);
	return topLevelTags;
}
function retrieveMetadata(doc) {
	var _a$5;
	var result = {};
	var attributes = (_a$5 = doc.querySelector("html")) === null || _a$5 === void 0 ? void 0 : _a$5.attributes;
	(attributes ? toArray(attributes) : []).forEach(function(attr) {
		result[attr.name] = attr.value;
	});
	toArray(doc.querySelectorAll("meta")).forEach(function(meta) {
		result[meta.name] = meta.content;
	});
	return result;
}
function retrieveHtmlStrings(clipboardData) {
	var _a$5;
	var rawHtml = (_a$5 = clipboardData.rawHtml) !== null && _a$5 !== void 0 ? _a$5 : "";
	var startIndex = rawHtml.indexOf(START_FRAGMENT);
	var endIndex = rawHtml.lastIndexOf(END_FRAGMENT);
	var htmlBefore = "";
	var htmlAfter = "";
	if (startIndex >= 0 && endIndex >= startIndex + START_FRAGMENT.length) {
		htmlBefore = rawHtml.substring(0, startIndex);
		htmlAfter = rawHtml.substring(endIndex + END_FRAGMENT.length);
		clipboardData.html = rawHtml.substring(startIndex + START_FRAGMENT.length, endIndex);
	} else clipboardData.html = rawHtml;
	return {
		htmlBefore,
		htmlAfter
	};
}
function checkBlockElements(doc) {
	return toArray(doc.body.querySelectorAll("*")).some(function(el) {
		return isNodeOfType(el, "ELEMENT_NODE") && isBlockElement(el);
	});
}
function paste(editor, clipboardData, pasteTypeOrGetter) {
	var _a$5;
	if (pasteTypeOrGetter === void 0) pasteTypeOrGetter = "normal";
	editor.focus();
	var isFirstPaste = false;
	if (!clipboardData.modelBeforePaste) {
		isFirstPaste = true;
		editor.formatContentModel(function(model) {
			clipboardData.modelBeforePaste = cloneModelForPaste(model);
			return false;
		});
	}
	var domCreator = editor.getDOMCreator();
	if (!domCreator.isBypassed && clipboardData.rawHtml) clipboardData.rawHtml = cleanHtmlComments(clipboardData.rawHtml);
	var doc = createDOMFromHtml(clipboardData.rawHtml, domCreator);
	var pasteType = typeof pasteTypeOrGetter == "function" ? pasteTypeOrGetter(doc, clipboardData) : pasteTypeOrGetter;
	var htmlFromClipboard = retrieveHtmlInfo(doc, clipboardData);
	var eventResult = generatePasteOptionFromPlugins(editor, clipboardData, createPasteFragment(editor.getDocument(), clipboardData, pasteType, (_a$5 = clipboardData.rawHtml == clipboardData.html ? doc : createDOMFromHtml(clipboardData.html, domCreator)) === null || _a$5 === void 0 ? void 0 : _a$5.body), htmlFromClipboard, pasteType);
	convertInlineCss(eventResult.fragment, htmlFromClipboard.globalCssRules);
	mergePasteContent(editor, eventResult, isFirstPaste);
}
function createDOMFromHtml(html$1, domCreator) {
	return html$1 ? domCreator.htmlToDOM(html$1) : null;
}
var CopyPastePlugin = function() {
	function CopyPastePlugin$1(option) {
		var _this = this;
		this.editor = null;
		this.disposer = null;
		this.onPaste = function(event$1) {
			if (_this.editor && isClipboardEvent(event$1)) {
				var editor_1 = _this.editor;
				var dataTransfer = event$1.clipboardData;
				if (shouldPreventDefaultPaste(dataTransfer, editor_1)) {
					event$1.preventDefault();
					extractClipboardItems(toArray(dataTransfer.items), _this.state.allowedCustomPasteType).then(function(clipboardData) {
						if (!editor_1.isDisposed()) paste(editor_1, clipboardData, _this.state.defaultPasteType);
					});
				}
			}
		};
		this.state = {
			allowedCustomPasteType: option.allowedCustomPasteType || [],
			tempDiv: null,
			defaultPasteType: option.defaultPasteType
		};
	}
	CopyPastePlugin$1.prototype.getName = function() {
		return "CopyPaste";
	};
	CopyPastePlugin$1.prototype.initialize = function(editor) {
		var _this = this;
		this.editor = editor;
		this.disposer = this.editor.attachDomEvent({
			paste: { beforeDispatch: function(e$1) {
				return _this.onPaste(e$1);
			} },
			copy: { beforeDispatch: function(e$1) {
				return _this.onCutCopy(e$1, false);
			} },
			cut: { beforeDispatch: function(e$1) {
				return _this.onCutCopy(e$1, true);
			} }
		});
	};
	CopyPastePlugin$1.prototype.dispose = function() {
		var _a$5;
		if (this.state.tempDiv) {
			(_a$5 = this.state.tempDiv.parentNode) === null || _a$5 === void 0 || _a$5.removeChild(this.state.tempDiv);
			this.state.tempDiv = null;
		}
		if (this.disposer) this.disposer();
		this.disposer = null;
		this.editor = null;
	};
	CopyPastePlugin$1.prototype.getState = function() {
		return this.state;
	};
	CopyPastePlugin$1.prototype.onCutCopy = function(event$1, isCut) {
		var _a$5, _b$1;
		if (!this.editor || !isClipboardEvent(event$1)) return;
		var textAndHtmlContent = getContentForCopy(this.editor, isCut, event$1);
		if (textAndHtmlContent) {
			var htmlContent = textAndHtmlContent.htmlContent, textContent = textAndHtmlContent.textContent;
			event$1.preventDefault();
			(_a$5 = event$1.clipboardData) === null || _a$5 === void 0 || _a$5.setData("text/html", htmlContent.innerHTML);
			(_b$1 = event$1.clipboardData) === null || _b$1 === void 0 || _b$1.setData("text/plain", textContent);
			if (isCut) this.editor.formatContentModel(function(model, context) {
				if (deleteSelection(model, [deleteEmptyList], context).deleteResult == "range") normalizeContentModel(model);
				return true;
			}, {
				apiName: "cut",
				changeSource: ChangeSource.Cut
			});
		}
	};
	return CopyPastePlugin$1;
}();
function isClipboardEvent(event$1) {
	return !!event$1.clipboardData;
}
function shouldPreventDefaultPaste(dataTransfer, editor) {
	if (!(dataTransfer === null || dataTransfer === void 0 ? void 0 : dataTransfer.items)) return false;
	if (!editor.getEnvironment().isAndroid) return true;
	return toArray(dataTransfer.items).some(function(item) {
		var type = item.type;
		var isNormalFile = item.kind === "file" && type !== "";
		var isText = type.indexOf("text/") === 0;
		return isNormalFile || isText;
	});
}
function createCopyPastePlugin(option) {
	return new CopyPastePlugin(option);
}
var EventTypeMap = {
	keydown: "keyDown",
	keyup: "keyUp",
	keypress: "keyPress"
};
var DOMEventPlugin = function() {
	function DOMEventPlugin$1(options, contentDiv) {
		var _this = this;
		this.editor = null;
		this.disposer = null;
		this.pointerEvent = null;
		this.timer = 0;
		this.onDragStart = function(e$1) {
			var dragEvent = e$1;
			var node = dragEvent.target;
			var element = isNodeOfType(node, "ELEMENT_NODE") ? node : node.parentElement;
			if (element && !element.isContentEditable) dragEvent.preventDefault();
		};
		this.onDrop = function() {
			var _a$5, _b$1;
			var doc = (_a$5 = _this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getDocument();
			(_b$1 = doc === null || doc === void 0 ? void 0 : doc.defaultView) === null || _b$1 === void 0 || _b$1.requestAnimationFrame(function() {
				if (_this.editor) {
					_this.editor.takeSnapshot();
					_this.editor.triggerEvent("contentChanged", { source: ChangeSource.Drop });
				}
			});
		};
		this.onScroll = function(e$1) {
			var _a$5;
			(_a$5 = _this.editor) === null || _a$5 === void 0 || _a$5.triggerEvent("scroll", {
				rawEvent: e$1,
				scrollContainer: _this.state.scrollContainer
			});
		};
		this.keyboardEventHandler = { beforeDispatch: function(event$1) {
			var _a$5, _b$1, _c$1;
			var eventType = EventTypeMap[event$1.type];
			if (isCharacterValue(event$1) || isCursorMovingKey(event$1)) event$1.stopPropagation();
			var isComposing = !((_c$1 = (_b$1 = (_a$5 = _this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getEnvironment()) === null || _b$1 === void 0 ? void 0 : _b$1.isAndroid) !== null && _c$1 !== void 0 ? _c$1 : false) && (event$1.isComposing || _this.state.isInIME);
			if (_this.editor && eventType && !isComposing) _this.editor.triggerEvent(eventType, { rawEvent: event$1 });
		} };
		this.inputEventHandler = { beforeDispatch: function(event$1) {
			var _a$5, _b$1, _c$1;
			event$1.stopPropagation();
			var isComposing = !((_c$1 = (_b$1 = (_a$5 = _this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getEnvironment()) === null || _b$1 === void 0 ? void 0 : _b$1.isAndroid) !== null && _c$1 !== void 0 ? _c$1 : false) && (event$1.isComposing || _this.state.isInIME);
			if (_this.editor && !isComposing) _this.editor.triggerEvent("input", { rawEvent: event$1 });
		} };
		this.onMouseDown = function(event$1) {
			if (_this.editor) {
				if (!_this.state.mouseUpEventListerAdded) {
					_this.editor.getDocument().addEventListener("mouseup", _this.onMouseUp, true);
					_this.state.mouseUpEventListerAdded = true;
					_this.state.mouseDownX = event$1.pageX;
					_this.state.mouseDownY = event$1.pageY;
				}
				_this.editor.triggerEvent("mouseDown", { rawEvent: event$1 });
				if (event$1.defaultPrevented) _this.pointerEvent = null;
				if (_this.pointerEvent) _this.editor.triggerEvent("pointerDown", {
					rawEvent: _this.pointerEvent,
					originalEvent: event$1
				});
			}
		};
		this.onMouseUp = function(rawEvent) {
			if (_this.editor) {
				_this.removeMouseUpEventListener();
				_this.editor.triggerEvent("mouseUp", {
					rawEvent,
					isClicking: _this.state.mouseDownX == rawEvent.pageX && _this.state.mouseDownY == rawEvent.pageY
				});
			}
			if (_this.pointerEvent) _this.pointerEvent = null;
		};
		this.onDoubleClick = function(event$1) {
			if (_this.editor) _this.editor.triggerEvent("doubleClick", { rawEvent: event$1 });
		};
		this.onCompositionStart = function() {
			_this.state.isInIME = true;
		};
		this.onCompositionEnd = function(rawEvent) {
			var _a$5;
			_this.state.isInIME = false;
			(_a$5 = _this.editor) === null || _a$5 === void 0 || _a$5.triggerEvent("compositionEnd", { rawEvent });
		};
		this.onPointerDown = function(e$1) {
			if (e$1.pointerType === "touch" || e$1.pointerType === "pen") _this.pointerEvent = e$1;
		};
		this.state = {
			isInIME: false,
			scrollContainer: options.scrollContainer || contentDiv,
			mouseDownX: null,
			mouseDownY: null,
			mouseUpEventListerAdded: false
		};
	}
	DOMEventPlugin$1.prototype.getName = function() {
		return "DOMEvent";
	};
	DOMEventPlugin$1.prototype.initialize = function(editor) {
		var _this = this;
		var _a$5, _b$1;
		this.editor = editor;
		var document$1 = this.editor.getDocument();
		var eventHandlers = {
			keypress: this.keyboardEventHandler,
			keydown: this.keyboardEventHandler,
			keyup: this.keyboardEventHandler,
			input: this.inputEventHandler,
			mousedown: { beforeDispatch: this.onMouseDown },
			dblclick: { beforeDispatch: function(event$1) {
				return _this.onDoubleClick(event$1);
			} },
			compositionstart: { beforeDispatch: this.onCompositionStart },
			compositionend: { beforeDispatch: this.onCompositionEnd },
			dragstart: { beforeDispatch: this.onDragStart },
			drop: { beforeDispatch: this.onDrop },
			pointerdown: { beforeDispatch: function(event$1) {
				return _this.onPointerDown(event$1);
			} }
		};
		this.disposer = this.editor.attachDomEvent(eventHandlers);
		this.state.scrollContainer.addEventListener("scroll", this.onScroll);
		(_a$5 = document$1.defaultView) === null || _a$5 === void 0 || _a$5.addEventListener("scroll", this.onScroll);
		(_b$1 = document$1.defaultView) === null || _b$1 === void 0 || _b$1.addEventListener("resize", this.onScroll);
	};
	DOMEventPlugin$1.prototype.dispose = function() {
		var _a$5, _b$1, _c$1, _d$1, _e$1;
		this.removeMouseUpEventListener();
		var document$1 = (_a$5 = this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getDocument();
		(_b$1 = document$1 === null || document$1 === void 0 ? void 0 : document$1.defaultView) === null || _b$1 === void 0 || _b$1.removeEventListener("resize", this.onScroll);
		(_c$1 = document$1 === null || document$1 === void 0 ? void 0 : document$1.defaultView) === null || _c$1 === void 0 || _c$1.removeEventListener("scroll", this.onScroll);
		this.state.scrollContainer.removeEventListener("scroll", this.onScroll);
		(_d$1 = this.disposer) === null || _d$1 === void 0 || _d$1.call(this);
		this.disposer = null;
		this.editor = null;
		this.pointerEvent = null;
		if (this.timer) {
			(_e$1 = document$1 === null || document$1 === void 0 ? void 0 : document$1.defaultView) === null || _e$1 === void 0 || _e$1.clearTimeout(this.timer);
			this.timer = 0;
		}
	};
	DOMEventPlugin$1.prototype.getState = function() {
		return this.state;
	};
	DOMEventPlugin$1.prototype.removeMouseUpEventListener = function() {
		if (this.editor && this.state.mouseUpEventListerAdded) {
			this.state.mouseUpEventListerAdded = false;
			this.editor.getDocument().removeEventListener("mouseup", this.onMouseUp, true);
		}
	};
	return DOMEventPlugin$1;
}();
function createDOMEventPlugin(option, contentDiv) {
	return new DOMEventPlugin(option, contentDiv);
}
function findAllEntities(group, entities) {
	group.blocks.forEach(function(block) {
		switch (block.blockType) {
			case "BlockGroup":
				findAllEntities(block, entities);
				break;
			case "Entity":
				entities.push({
					entity: block,
					operation: "newEntity"
				});
				break;
			case "Paragraph":
				block.segments.forEach(function(segment) {
					switch (segment.segmentType) {
						case "Entity":
							entities.push({
								entity: segment,
								operation: "newEntity"
							});
							break;
						case "General":
							findAllEntities(segment, entities);
							break;
					}
				});
				break;
			case "Table":
				block.rows.forEach(function(row) {
					return row.cells.forEach(function(cell) {
						return findAllEntities(cell, entities);
					});
				});
				break;
		}
	});
}
function adjustSelectionAroundEntity(editor, key, shiftKey) {
	var _a$5;
	var selection = editor.isDisposed() ? null : editor.getDOMSelection();
	if (!selection || selection.type != "range") return;
	var range = selection.range, isReverted = selection.isReverted;
	var anchorNode = isReverted ? range.startContainer : range.endContainer;
	var offset = isReverted ? range.startOffset : range.endOffset;
	var delimiter = isNodeOfType(anchorNode, "ELEMENT_NODE") ? anchorNode : anchorNode.parentElement;
	var isRtl = delimiter && ((_a$5 = editor.getDocument().defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getComputedStyle(delimiter).direction) == "rtl";
	var movingBefore = key == "ArrowLeft" != !!isRtl;
	if (delimiter && (isEntityDelimiter(delimiter, !movingBefore) && (movingBefore && offset == 0 || !movingBefore && offset == 1) || isBlockEntityContainer(delimiter))) editor.formatContentModel(function(model) {
		var _a$6, _b$1;
		var allSel = getSelectedSegmentsAndParagraphs(model, false, true);
		var sel = allSel[isReverted ? 0 : allSel.length - 1];
		var index$1 = (_b$1 = (_a$6 = sel === null || sel === void 0 ? void 0 : sel[1]) === null || _a$6 === void 0 ? void 0 : _a$6.segments.indexOf(sel[0])) !== null && _b$1 !== void 0 ? _b$1 : -1;
		if (sel && sel[1] && index$1 >= 0) {
			var _c$1 = __read(sel, 3), segment = _c$1[0], paragraph = _c$1[1], path = _c$1[2];
			var isShrinking = shiftKey && !range.collapsed && movingBefore != !!isReverted;
			var pairedDelimiter = findPairedDelimiter(isShrinking ? segment : paragraph.segments[movingBefore ? index$1 - 1 : index$1 + 1], path, paragraph, movingBefore);
			if (pairedDelimiter) {
				var newRange = getNewRange(range, isShrinking, movingBefore, pairedDelimiter, shiftKey);
				editor.setDOMSelection({
					type: "range",
					range: newRange,
					isReverted: newRange.collapsed ? false : isReverted
				});
			}
		}
		return false;
	});
}
function getNewRange(originalRange, isShrinking, movingBefore, pairedDelimiter, shiftKey) {
	var newRange = originalRange.cloneRange();
	if (isShrinking) if (movingBefore) newRange.setEndBefore(pairedDelimiter);
	else newRange.setStartAfter(pairedDelimiter);
	else {
		if (movingBefore) newRange.setStartBefore(pairedDelimiter);
		else newRange.setEndAfter(pairedDelimiter);
		if (!shiftKey) if (movingBefore) newRange.setEndBefore(pairedDelimiter);
		else newRange.setStartAfter(pairedDelimiter);
	}
	return newRange;
}
function findPairedDelimiter(entitySegment, path, paragraph, movingBefore) {
	var entity = null;
	if ((entitySegment === null || entitySegment === void 0 ? void 0 : entitySegment.segmentType) == "Entity") entity = entitySegment;
	else {
		var blocks = path[0].blocks;
		var paraIndex = blocks.indexOf(paragraph);
		var entityBlock = paraIndex >= 0 ? blocks[movingBefore ? paraIndex - 1 : paraIndex + 1] : null;
		if ((entityBlock === null || entityBlock === void 0 ? void 0 : entityBlock.blockType) == "Entity") entity = entityBlock;
	}
	var pairedDelimiter = entity ? movingBefore ? entity.wrapper.previousElementSibling : entity.wrapper.nextElementSibling : null;
	return isNodeOfType(pairedDelimiter, "ELEMENT_NODE") && isEntityDelimiter(pairedDelimiter, movingBefore) ? pairedDelimiter : null;
}
var HTML_VOID_ELEMENTS = [
	"AREA",
	"BASE",
	"BR",
	"COL",
	"COMMAND",
	"EMBED",
	"HR",
	"IMG",
	"INPUT",
	"KEYGEN",
	"LINK",
	"META",
	"PARAM",
	"SOURCE",
	"TRACK",
	"WBR"
];
function normalizePos(node, offset) {
	var _a$5, _b$1, _c$1, _d$1;
	var len = isNodeOfType(node, "TEXT_NODE") ? (_b$1 = (_a$5 = node.nodeValue) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0 : node.childNodes.length;
	offset = Math.max(Math.min(offset, len), 0);
	while (node === null || node === void 0 ? void 0 : node.lastChild) if (offset >= node.childNodes.length) {
		node = node.lastChild;
		offset = isNodeOfType(node, "TEXT_NODE") ? (_d$1 = (_c$1 = node.nodeValue) === null || _c$1 === void 0 ? void 0 : _c$1.length) !== null && _d$1 !== void 0 ? _d$1 : 0 : node.childNodes.length;
	} else {
		var nextNode = node.childNodes[offset];
		if (isNodeOfType(nextNode, "ELEMENT_NODE") && HTML_VOID_ELEMENTS.indexOf(nextNode.tagName) >= 0) break;
		else {
			node = node.childNodes[offset];
			offset = 0;
		}
	}
	return {
		node,
		offset
	};
}
var DelimiterBefore = "entityDelimiterBefore";
var DelimiterAfter = "entityDelimiterAfter";
var DelimiterSelector = "." + DelimiterAfter + ",." + DelimiterBefore;
var ZeroWidthSpace = "​";
var InlineEntitySelector = "span._Entity";
function preventTypeInDelimiter(node, editor) {
	var entitySibling = node.classList.contains(DelimiterAfter) ? node.previousElementSibling : node.nextElementSibling;
	if (entitySibling && isEntityElement(entitySibling)) {
		removeInvalidDelimiters([entitySibling.previousElementSibling, entitySibling.nextElementSibling].filter(function(element) {
			return !!element;
		}));
		editor.formatContentModel(function(model, context) {
			iterateSelections(model, function(_path, _tableContext, block, _segments) {
				if ((block === null || block === void 0 ? void 0 : block.blockType) == "Paragraph") block.segments.forEach(function(segment) {
					if (segment.segmentType == "Text" && segment.text.indexOf(ZeroWidthSpace) >= 0) mutateSegment(block, segment, function(segment$1) {
						segment$1.text = segment$1.text.replace(ZeroWidthSpace, "");
					});
				});
			});
			context.skipUndoSnapshot = true;
			return true;
		});
	}
}
function addDelimitersIfNeeded(nodes, format) {
	if (nodes.length > 0) {
		var context_1 = createModelToDomContext();
		nodes.forEach(function(node) {
			if (isNodeOfType(node, "ELEMENT_NODE") && isEntityElement(node) && !node.isContentEditable) addDelimiters(node.ownerDocument, node, format, context_1);
		});
	}
}
function removeNode(el) {
	var _a$5;
	(_a$5 = el === null || el === void 0 ? void 0 : el.parentElement) === null || _a$5 === void 0 || _a$5.removeChild(el);
}
function removeInvalidDelimiters(nodes) {
	nodes.forEach(function(node) {
		if (!isNodeOfType(node, "ELEMENT_NODE")) return;
		if (isEntityDelimiter(node)) {
			var sibling$1 = node.classList.contains(DelimiterBefore) ? node.nextElementSibling : node.previousElementSibling;
			if (!(isNodeOfType(sibling$1, "ELEMENT_NODE") && isEntityElement(sibling$1))) removeNode(node);
		} else removeDelimiterAttr(node);
	});
}
function removeDelimiterAttr(node, checkEntity) {
	if (checkEntity === void 0) checkEntity = true;
	if (!node) return;
	var entitySibling = node.classList.contains(DelimiterAfter) ? node.previousElementSibling : node.nextElementSibling;
	if (checkEntity && entitySibling && isEntityElement(entitySibling)) return;
	node.classList.remove(DelimiterAfter, DelimiterBefore);
	node.normalize();
	node.childNodes.forEach(function(cn) {
		var _a$5, _b$1;
		var index$1 = (_b$1 = (_a$5 = cn.textContent) === null || _a$5 === void 0 ? void 0 : _a$5.indexOf(ZeroWidthSpace)) !== null && _b$1 !== void 0 ? _b$1 : -1;
		if (index$1 >= 0) {
			var range = new Range();
			range.setStart(cn, index$1);
			range.setEnd(cn, index$1 + 1);
			range.deleteContents();
		}
	});
}
function getFocusedElement(selection, existingTextInDelimiter) {
	var _a$5, _b$1, _c$1, _d$1, _e$1;
	var range = selection.range, isReverted = selection.isReverted;
	var node = isReverted ? range.startContainer : range.endContainer;
	var offset = isReverted ? range.startOffset : range.endOffset;
	if (node) {
		var pos = normalizePos(node, offset);
		node = pos.node;
		offset = pos.offset;
	}
	if (!isNodeOfType(node, "ELEMENT_NODE")) {
		var textToCheck = existingTextInDelimiter ? ZeroWidthSpace + existingTextInDelimiter : ZeroWidthSpace;
		if (node.textContent != textToCheck && (node.textContent || "").length == offset) node = (_c$1 = (_a$5 = node.nextSibling) !== null && _a$5 !== void 0 ? _a$5 : (_b$1 = node.parentElement) === null || _b$1 === void 0 ? void 0 : _b$1.closest(DelimiterSelector)) !== null && _c$1 !== void 0 ? _c$1 : null;
		else node = (_e$1 = (_d$1 = node === null || node === void 0 ? void 0 : node.parentElement) === null || _d$1 === void 0 ? void 0 : _d$1.closest(DelimiterSelector)) !== null && _e$1 !== void 0 ? _e$1 : null;
	} else node = node.childNodes.length == offset ? node : node.childNodes.item(offset);
	if (node && !node.hasChildNodes()) node = node.nextSibling;
	return isNodeOfType(node, "ELEMENT_NODE") ? node : null;
}
function handleDelimiterContentChangedEvent(editor) {
	var helper = editor.getDOMHelper();
	removeInvalidDelimiters(helper.queryElements(DelimiterSelector));
	addDelimitersIfNeeded(helper.queryElements(InlineEntitySelector), editor.getPendingFormat());
}
function handleCompositionEndEvent(editor, event$1) {
	var selection = editor.getDOMSelection();
	if ((selection === null || selection === void 0 ? void 0 : selection.type) == "range" && selection.range.collapsed) {
		var node = getFocusedElement(selection, event$1.rawEvent.data);
		if ((node === null || node === void 0 ? void 0 : node.firstChild) && isNodeOfType(node.firstChild, "TEXT_NODE") && node.matches(DelimiterSelector) && node.textContent == ZeroWidthSpace + event$1.rawEvent.data) preventTypeInDelimiter(node, editor);
	}
}
function handleDelimiterKeyDownEvent(editor, event$1) {
	var _a$5;
	var selection = editor.getDOMSelection();
	if (!selection || selection.type != "range") return;
	var rawEvent = event$1.rawEvent;
	var range = selection.range;
	var key = rawEvent.key;
	switch (key) {
		case "Enter":
			var helper = editor.getDOMHelper();
			var entity = findClosestEntityWrapper(range.startContainer, helper);
			if (entity && isNodeOfType(entity, "ELEMENT_NODE") && helper.isNodeInEditor(entity)) triggerEntityEventOnEnter(editor, entity, rawEvent);
			break;
		case "ArrowLeft":
		case "ArrowRight":
			if (!rawEvent.altKey && !rawEvent.ctrlKey && !rawEvent.metaKey) (_a$5 = editor.getDocument().defaultView) === null || _a$5 === void 0 || _a$5.requestAnimationFrame(function() {
				adjustSelectionAroundEntity(editor, key, rawEvent.shiftKey);
			});
			break;
		default:
			if (isCharacterValue(rawEvent) && range.collapsed) handleInputOnDelimiter(editor, range, getFocusedElement(selection), rawEvent);
			break;
	}
}
function handleInputOnDelimiter(editor, range, focusedNode, rawEvent) {
	var _a$5;
	var helper = editor.getDOMHelper();
	if (focusedNode && isEntityDelimiter(focusedNode) && helper.isNodeInEditor(focusedNode)) {
		var blockEntityContainer = findClosestBlockEntityContainer(focusedNode, helper);
		var isEnter = rawEvent.key === "Enter";
		if (blockEntityContainer && helper.isNodeInEditor(blockEntityContainer)) {
			if (focusedNode.classList.contains(DelimiterAfter)) range.setStartAfter(blockEntityContainer);
			else range.setStartBefore(blockEntityContainer);
			range.collapse(true);
			if (isEnter) rawEvent.preventDefault();
			editor.formatContentModel(handleKeyDownInBlockDelimiter, { selectionOverride: {
				type: "range",
				isReverted: false,
				range
			} });
		} else if (isEnter) editor.formatContentModel(function(model, context) {
			var result = handleEnterInlineEntity(model, context);
			if (result) rawEvent.preventDefault();
			return result;
		});
		else {
			editor.takeSnapshot();
			(_a$5 = editor.getDocument().defaultView) === null || _a$5 === void 0 || _a$5.requestAnimationFrame(function() {
				return preventTypeInDelimiter(focusedNode, editor);
			});
		}
	}
}
var handleKeyDownInBlockDelimiter = function(model, context) {
	iterateSelections(model, function(_path, _tableContext, readonlyBlock) {
		if ((readonlyBlock === null || readonlyBlock === void 0 ? void 0 : readonlyBlock.blockType) == "Paragraph") {
			var block = mutateBlock(readonlyBlock);
			delete block.isImplicit;
			var selectionMarker = block.segments.find(function(w$1) {
				return w$1.segmentType == "SelectionMarker";
			});
			if ((selectionMarker === null || selectionMarker === void 0 ? void 0 : selectionMarker.segmentType) == "SelectionMarker") {
				block.segmentFormat = __assign({}, selectionMarker.format);
				context.newPendingFormat = __assign({}, selectionMarker.format);
			}
			block.segments.unshift(createBr());
		}
	});
	return true;
};
var handleEnterInlineEntity = function(model) {
	var _a$5;
	var readonlySelectionBlock;
	var selectionBlockParent;
	iterateSelections(model, function(path, _tableContext, block) {
		if ((block === null || block === void 0 ? void 0 : block.blockType) == "Paragraph") {
			readonlySelectionBlock = block;
			selectionBlockParent = path[0];
		}
	});
	if ((selectionBlockParent === null || selectionBlockParent === void 0 ? void 0 : selectionBlockParent.blockGroupType) == "ListItem") return false;
	if (readonlySelectionBlock && selectionBlockParent) {
		var markerIndex = readonlySelectionBlock.segments.findIndex(function(segment) {
			return segment.segmentType == "SelectionMarker";
		});
		if (markerIndex >= 0) {
			var selectionBlock = mutateBlock(readonlySelectionBlock);
			var segmentsAfterMarker = selectionBlock.segments.splice(markerIndex);
			var newPara = createParagraph(false, selectionBlock.format, selectionBlock.segmentFormat, selectionBlock.decorator);
			if (selectionBlock.segments.every(function(x$1) {
				return x$1.segmentType == "SelectionMarker" || x$1.segmentType == "Br";
			}) || segmentsAfterMarker.every(function(x$1) {
				return x$1.segmentType == "SelectionMarker";
			})) newPara.segments.push(createBr(selectionBlock.format));
			(_a$5 = newPara.segments).push.apply(_a$5, __spreadArray([], __read(segmentsAfterMarker), false));
			var selectionBlockIndex = selectionBlockParent.blocks.indexOf(selectionBlock);
			if (selectionBlockIndex >= 0) mutateBlock(selectionBlockParent).blocks.splice(selectionBlockIndex + 1, 0, newPara);
		}
	}
	return true;
};
var triggerEntityEventOnEnter = function(editor, wrapper, rawEvent) {
	var format = parseEntityFormat(wrapper);
	if (format.id && format.entityType && !format.isFakeEntity) editor.triggerEvent("entityOperation", {
		operation: "click",
		entity: {
			id: format.id,
			type: format.entityType,
			isReadonly: !!format.isReadonly,
			wrapper
		},
		rawEvent
	});
};
var ENTITY_ID_REGEX = /_(\d{1,8})$/;
var EntityPlugin = function() {
	function EntityPlugin$1() {
		this.editor = null;
		this.state = { entityMap: {} };
	}
	EntityPlugin$1.prototype.getName = function() {
		return "Entity";
	};
	EntityPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
	};
	EntityPlugin$1.prototype.dispose = function() {
		this.editor = null;
		this.state.entityMap = {};
	};
	EntityPlugin$1.prototype.getState = function() {
		return this.state;
	};
	EntityPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (this.editor) switch (event$1.eventType) {
			case "mouseUp":
				this.handleMouseUpEvent(this.editor, event$1);
				break;
			case "contentChanged":
				this.handleContentChangedEvent(this.editor, event$1);
				break;
			case "keyDown":
				handleDelimiterKeyDownEvent(this.editor, event$1);
				break;
			case "compositionEnd":
				handleCompositionEndEvent(this.editor, event$1);
				break;
			case "editorReady":
				this.handleContentChangedEvent(this.editor);
				break;
			case "extractContentWithDom":
				this.handleExtractContentWithDomEvent(this.editor, event$1.clonedRoot);
				break;
		}
	};
	EntityPlugin$1.prototype.handleMouseUpEvent = function(editor, event$1) {
		var rawEvent = event$1.rawEvent, isClicking = event$1.isClicking;
		var node = rawEvent.target;
		if (isClicking && this.editor) while (node && this.editor.getDOMHelper().isNodeInEditor(node)) if (isEntityElement(node)) {
			this.triggerEvent(editor, node, "click", rawEvent);
			break;
		} else node = node.parentNode;
	};
	EntityPlugin$1.prototype.handleContentChangedEvent = function(editor, event$1) {
		var _this = this;
		var _a$5;
		var modifiedEntities = (_a$5 = event$1 === null || event$1 === void 0 ? void 0 : event$1.changedEntities) !== null && _a$5 !== void 0 ? _a$5 : this.getChangedEntities(editor);
		var entityStates = event$1 === null || event$1 === void 0 ? void 0 : event$1.entityStates;
		modifiedEntities.forEach(function(entry) {
			var entity = entry.entity, operation = entry.operation, rawEvent = entry.rawEvent;
			var _a$6 = entity.entityFormat, id = _a$6.id, entityType = _a$6.entityType, isFakeEntity = _a$6.isFakeEntity, wrapper = entity.wrapper;
			if (entityType && !isFakeEntity) {
				if (operation == "newEntity") {
					entity.entityFormat.id = _this.ensureUniqueId(entityType, id !== null && id !== void 0 ? id : "", wrapper);
					wrapper.className = generateEntityClassNames(entity.entityFormat);
					if (entity.entityFormat.isReadonly) wrapper.contentEditable = "false";
					var eventResult = _this.triggerEvent(editor, wrapper, operation, rawEvent);
					_this.state.entityMap[entity.entityFormat.id] = {
						element: wrapper,
						canPersist: eventResult === null || eventResult === void 0 ? void 0 : eventResult.shouldPersist
					};
					if (editor.isDarkMode()) transformColor(wrapper, true, "lightToDark", editor.getColorManager());
				} else if (id) {
					var mapEntry = _this.state.entityMap[id];
					if (mapEntry) mapEntry.isDeleted = true;
					_this.triggerEvent(editor, wrapper, operation, rawEvent);
				}
			}
		});
		entityStates === null || entityStates === void 0 || entityStates.forEach(function(entityState) {
			var _a$6;
			var id = entityState.id, state$1 = entityState.state;
			var wrapper = (_a$6 = _this.state.entityMap[id]) === null || _a$6 === void 0 ? void 0 : _a$6.element;
			if (wrapper) _this.triggerEvent(editor, wrapper, "updateEntityState", void 0, state$1);
		});
		handleDelimiterContentChangedEvent(editor);
	};
	EntityPlugin$1.prototype.getChangedEntities = function(editor) {
		var _this = this;
		var result = [];
		editor.formatContentModel(function(model) {
			findAllEntities(model, result);
			return false;
		});
		getObjectKeys(this.state.entityMap).forEach(function(id) {
			var entry = _this.state.entityMap[id];
			if (!entry.isDeleted) {
				var index$1 = result.findIndex(function(x$1) {
					return x$1.operation == "newEntity" && x$1.entity.entityFormat.id == id && x$1.entity.wrapper == entry.element;
				});
				if (index$1 >= 0) result.splice(index$1, 1);
				else {
					var format = parseEntityFormat(entry.element);
					if (!format.isFakeEntity) {
						var entity = createEntity(entry.element, format.isReadonly, {}, format.entityType, format.id);
						result.push({
							entity,
							operation: "overwrite"
						});
					}
				}
			}
		});
		return result;
	};
	EntityPlugin$1.prototype.handleExtractContentWithDomEvent = function(editor, root$12) {
		var _this = this;
		getAllEntityWrappers(root$12).forEach(function(element) {
			element.removeAttribute("contentEditable");
			_this.triggerEvent(editor, element, "replaceTemporaryContent");
		});
	};
	EntityPlugin$1.prototype.triggerEvent = function(editor, wrapper, operation, rawEvent, state$1) {
		var format = parseEntityFormat(wrapper);
		return format.id && format.entityType && !format.isFakeEntity ? editor.triggerEvent("entityOperation", {
			operation,
			rawEvent,
			entity: {
				id: format.id,
				type: format.entityType,
				isReadonly: !!format.isReadonly,
				wrapper
			},
			state: operation == "updateEntityState" ? state$1 : void 0,
			shouldPersist: operation == "newEntity" ? true : void 0
		}) : null;
	};
	EntityPlugin$1.prototype.ensureUniqueId = function(type, id, wrapper) {
		var match = ENTITY_ID_REGEX.exec(id);
		var baseId = (match ? id.substr(0, id.length - match[0].length) : id) || type;
		var newId = "";
		for (var num = match && parseInt(match[1]) || 0;; num++) {
			newId = num > 0 ? baseId + "_" + num : baseId;
			var item = this.state.entityMap[newId];
			if (!item || item.element == wrapper) break;
		}
		return newId;
	};
	return EntityPlugin$1;
}();
function createEntityPlugin() {
	return new EntityPlugin();
}
function applyDefaultFormat(editor, defaultFormat) {
	var selection = editor.getDOMSelection();
	if ((selection === null || selection === void 0 ? void 0 : selection.type) == "range" && selection.range.collapsed) editor.formatContentModel(function(model, context) {
		iterateSelections(model, function(path, _, paragraph, segments) {
			var marker = segments === null || segments === void 0 ? void 0 : segments[0];
			if ((paragraph === null || paragraph === void 0 ? void 0 : paragraph.blockType) == "Paragraph" && (marker === null || marker === void 0 ? void 0 : marker.segmentType) == "SelectionMarker") {
				var blocks = path[0].blocks;
				var blockCount = blocks.length;
				var blockIndex = blocks.indexOf(paragraph);
				if (paragraph.isImplicit && paragraph.segments.length == 1 && paragraph.segments[0] == marker && blockCount > 0 && blockIndex == blockCount - 1) {
					var previousBlock = blocks[blockIndex - 1];
					if ((previousBlock === null || previousBlock === void 0 ? void 0 : previousBlock.blockType) != "Paragraph") context.newPendingFormat = getNewPendingFormat(editor, defaultFormat, marker.format);
				} else if (paragraph.segments.every(function(x$1) {
					return x$1.segmentType != "Text";
				})) context.newPendingFormat = getNewPendingFormat(editor, defaultFormat, marker.format);
			}
			return true;
		});
		return false;
	});
}
function getNewPendingFormat(editor, defaultFormat, markerFormat) {
	return __assign(__assign(__assign({}, defaultFormat), editor.getPendingFormat()), markerFormat);
}
var ANSI_SPACE = " ";
var NON_BREAK_SPACE = "\xA0";
function applyPendingFormat(editor, data, segmentFormat, paragraphFormat) {
	var isChanged = false;
	editor.formatContentModel(function(model, context) {
		iterateSelections(model, function(_, __, block, segments) {
			if ((block === null || block === void 0 ? void 0 : block.blockType) == "Paragraph" && (segments === null || segments === void 0 ? void 0 : segments.length) == 1 && segments[0].segmentType == "SelectionMarker") {
				var marker = segments[0];
				var index_1 = block.segments.indexOf(marker);
				var previousSegment_1 = block.segments[index_1 - 1];
				if ((previousSegment_1 === null || previousSegment_1 === void 0 ? void 0 : previousSegment_1.segmentType) == "Text") {
					var text_1 = previousSegment_1.text;
					var subStr = text_1.substr(-data.length, data.length);
					if (subStr == data || data == ANSI_SPACE && subStr == NON_BREAK_SPACE) {
						if (segmentFormat && !isSubFormatIncluded(previousSegment_1.format, segmentFormat)) {
							mutateSegment(block, previousSegment_1, function(previousSegment) {
								previousSegment.text = text_1.substring(0, text_1.length - data.length);
							});
							mutateSegment(block, marker, function(marker$1, block$1) {
								marker$1.format = __assign({}, segmentFormat);
								var newText = createText(data == ANSI_SPACE ? NON_BREAK_SPACE : data, __assign(__assign({}, previousSegment_1.format), segmentFormat));
								block$1.segments.splice(index_1, 0, newText);
								setParagraphNotImplicit(block$1);
							});
							isChanged = true;
						}
						if (paragraphFormat && !isSubFormatIncluded(block.format, paragraphFormat)) {
							var mutableParagraph = mutateBlock(block);
							Object.assign(mutableParagraph.format, paragraphFormat);
							isChanged = true;
						}
					}
				}
			}
			return true;
		});
		if (isChanged) {
			normalizeContentModel(model);
			context.skipUndoSnapshot = true;
		}
		return isChanged;
	}, { apiName: "applyPendingFormat" });
}
function isSubFormatIncluded(containerFormat, subFormat) {
	var keys = getObjectKeys(subFormat);
	var result = true;
	keys.forEach(function(key) {
		if (containerFormat[key] !== subFormat[key]) result = false;
	});
	return result;
}
var ProcessKey = "Process";
var UnidentifiedKey = "Unidentified";
var DefaultStyleKeyMap = {
	backgroundColor: "backgroundColor",
	textColor: "color",
	fontFamily: "fontFamily",
	fontSize: "fontSize"
};
var FormatPlugin = function() {
	function FormatPlugin$1(option) {
		var _this = this;
		this.editor = null;
		this.lastCheckedNode = null;
		this.state = {
			defaultFormat: __assign({}, option.defaultSegmentFormat),
			pendingFormat: null
		};
		var defaultFormat = this.state.defaultFormat;
		if (defaultFormat.fontFamily) defaultFormat.fontFamily = normalizeFontFamily(defaultFormat.fontFamily);
		this.defaultFormatKeys = /* @__PURE__ */ new Set();
		getObjectKeys(DefaultStyleKeyMap).forEach(function(key) {
			if (_this.state.defaultFormat[key]) _this.defaultFormatKeys.add(DefaultStyleKeyMap[key]);
		});
	}
	FormatPlugin$1.prototype.getName = function() {
		return "Format";
	};
	FormatPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		this.state.defaultFormat = normalizeSegmentFormat(this.state.defaultFormat, editor.getEnvironment());
	};
	FormatPlugin$1.prototype.dispose = function() {
		this.editor = null;
	};
	FormatPlugin$1.prototype.getState = function() {
		return this.state;
	};
	FormatPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (!this.editor) return;
		switch (event$1.eventType) {
			case "input":
				this.checkAndApplyPendingFormat(event$1.rawEvent.data);
				break;
			case "compositionEnd":
				this.checkAndApplyPendingFormat(event$1.rawEvent.data);
				break;
			case "keyDown":
				var isAndroidIME = this.editor.getEnvironment().isAndroid && event$1.rawEvent.key == UnidentifiedKey;
				if (isCursorMovingKey(event$1.rawEvent)) {
					this.clearPendingFormat();
					this.lastCheckedNode = null;
				} else if (this.defaultFormatKeys.size > 0 && (isAndroidIME || isCharacterValue(event$1.rawEvent) || event$1.rawEvent.key == ProcessKey) && this.shouldApplyDefaultFormat(this.editor)) applyDefaultFormat(this.editor, this.state.defaultFormat);
				break;
			case "mouseUp":
			case "contentChanged":
				this.lastCheckedNode = null;
				if (!this.canApplyPendingFormat()) this.clearPendingFormat();
				break;
		}
	};
	FormatPlugin$1.prototype.checkAndApplyPendingFormat = function(data) {
		if (this.editor && data && this.state.pendingFormat) {
			applyPendingFormat(this.editor, data, this.state.pendingFormat.format, this.state.pendingFormat.paragraphFormat);
			this.clearPendingFormat();
		}
	};
	FormatPlugin$1.prototype.clearPendingFormat = function() {
		this.state.pendingFormat = null;
	};
	FormatPlugin$1.prototype.canApplyPendingFormat = function() {
		var result = false;
		if (this.state.pendingFormat && this.editor) {
			var selection = this.editor.getDOMSelection();
			var range = (selection === null || selection === void 0 ? void 0 : selection.type) == "range" && selection.range.collapsed ? selection.range : null;
			var _a$5 = this.state.pendingFormat.insertPoint, node = _a$5.node, offset = _a$5.offset;
			if (range && range.startContainer == node && range.startOffset == offset) result = true;
		}
		return result;
	};
	FormatPlugin$1.prototype.shouldApplyDefaultFormat = function(editor) {
		var _a$5, _b$1;
		var selection = editor.getDOMSelection();
		var range = (selection === null || selection === void 0 ? void 0 : selection.type) == "range" ? selection.range : null;
		var posContainer = (_a$5 = range === null || range === void 0 ? void 0 : range.startContainer) !== null && _a$5 !== void 0 ? _a$5 : null;
		if (posContainer && posContainer != this.lastCheckedNode) {
			this.lastCheckedNode = posContainer;
			var domHelper = editor.getDOMHelper();
			var element = isNodeOfType(posContainer, "ELEMENT_NODE") ? posContainer : posContainer.parentElement;
			var foundFormatKeys_1 = /* @__PURE__ */ new Set();
			var _loop_1 = function() {
				if ((_b$1 = element.getAttribute) === null || _b$1 === void 0 ? void 0 : _b$1.call(element, "style")) {
					var style_1 = element.style;
					this_1.defaultFormatKeys.forEach(function(key) {
						if (style_1[key]) foundFormatKeys_1.add(key);
					});
					if (foundFormatKeys_1.size == this_1.defaultFormatKeys.size) return { value: false };
				}
				if (isBlockElement(element)) return "break";
				element = element.parentElement;
			};
			var this_1 = this;
			while ((element === null || element === void 0 ? void 0 : element.parentElement) && domHelper.isNodeInEditor(element.parentElement)) {
				var state_1 = _loop_1();
				if (typeof state_1 === "object") return state_1.value;
				if (state_1 === "break") break;
			}
			return true;
		} else return false;
	};
	return FormatPlugin$1;
}();
function createFormatPlugin(option) {
	return new FormatPlugin(option);
}
var ContentEditableAttributeName = "contenteditable";
var DefaultTextColor = "#000000";
var DefaultBackColor = "#ffffff";
var LifecyclePlugin = function() {
	function LifecyclePlugin$1(options, contentDiv) {
		var _this = this;
		this.editor = null;
		this.initializer = null;
		this.disposer = null;
		if (contentDiv.getAttribute(ContentEditableAttributeName) === null) {
			this.initializer = function() {
				contentDiv.contentEditable = "true";
				contentDiv.style.userSelect = "text";
			};
			this.disposer = function() {
				contentDiv.style.userSelect = "";
				contentDiv.removeAttribute(ContentEditableAttributeName);
			};
		}
		this.adjustColor = options.doNotAdjustEditorColor ? function() {} : function() {
			_this.adjustContainerColor(contentDiv);
		};
		this.state = {
			isDarkMode: !!options.inDarkMode,
			shadowEditFragment: null,
			styleElements: {},
			announcerStringGetter: options.announcerStringGetter
		};
	}
	LifecyclePlugin$1.prototype.getName = function() {
		return "Lifecycle";
	};
	LifecyclePlugin$1.prototype.initialize = function(editor) {
		var _a$5, _b$1;
		this.editor = editor;
		(_a$5 = this.initializer) === null || _a$5 === void 0 || _a$5.call(this);
		this.adjustColor();
		var rewriteFromModel = (_b$1 = this.state.rewriteFromModel) !== null && _b$1 !== void 0 ? _b$1 : {
			addedBlockElements: [],
			removedBlockElements: []
		};
		this.editor.triggerEvent("editorReady", rewriteFromModel, true);
		delete this.state.rewriteFromModel;
		this.state.announceContainer = createAriaLiveElement(editor.getDocument());
	};
	LifecyclePlugin$1.prototype.dispose = function() {
		var _this = this;
		var _a$5, _b$1;
		(_a$5 = this.editor) === null || _a$5 === void 0 || _a$5.triggerEvent("beforeDispose", {}, true);
		getObjectKeys(this.state.styleElements).forEach(function(key) {
			var _a$6;
			var element = _this.state.styleElements[key];
			(_a$6 = element.parentElement) === null || _a$6 === void 0 || _a$6.removeChild(element);
			delete _this.state.styleElements[key];
		});
		var announceContainer = this.state.announceContainer;
		if (announceContainer) {
			(_b$1 = announceContainer.parentElement) === null || _b$1 === void 0 || _b$1.removeChild(announceContainer);
			delete this.state.announceContainer;
		}
		if (this.disposer) {
			this.disposer();
			this.disposer = null;
			this.initializer = null;
		}
		this.editor = null;
	};
	LifecyclePlugin$1.prototype.getState = function() {
		return this.state;
	};
	LifecyclePlugin$1.prototype.onPluginEvent = function(event$1) {
		if (event$1.eventType == "contentChanged" && (event$1.source == ChangeSource.SwitchToDarkMode || event$1.source == ChangeSource.SwitchToLightMode)) this.adjustColor();
	};
	LifecyclePlugin$1.prototype.adjustContainerColor = function(contentDiv) {
		if (this.editor) {
			var isDarkMode = this.state.isDarkMode;
			var darkColorHandler = this.editor.getColorManager();
			setColor(contentDiv, DefaultTextColor, false, isDarkMode, darkColorHandler);
			setColor(contentDiv, DefaultBackColor, true, isDarkMode, darkColorHandler);
		}
	};
	return LifecyclePlugin$1;
}();
function createLifecyclePlugin(option, contentDiv) {
	return new LifecyclePlugin(option, contentDiv);
}
var TableCellSelector = "TH,TD";
function findCoordinate(parsedTable, element, domHelper) {
	var td = domHelper.findClosestElementAncestor(element, TableCellSelector);
	var result = null;
	if (td) parsedTable.some(function(row, rowIndex) {
		var colIndex = td ? row.indexOf(td) : -1;
		return result = colIndex >= 0 ? {
			row: rowIndex,
			col: colIndex
		} : null;
	});
	if (!result) parsedTable.some(function(row, rowIndex) {
		var colIndex = row.findIndex(function(cell) {
			return typeof cell == "object" && cell.contains(element);
		});
		return result = colIndex >= 0 ? {
			row: rowIndex,
			col: colIndex
		} : null;
	});
	return result;
}
function isSingleImageInSelection(selection) {
	var _a$5 = getProps(selection), startNode = _a$5.startNode, endNode = _a$5.endNode, startOffset = _a$5.startOffset, endOffset = _a$5.endOffset;
	var max = Math.max(startOffset, endOffset);
	var min = Math.min(startOffset, endOffset);
	if (startNode && endNode && startNode == endNode && max - min == 1) {
		var node = startNode === null || startNode === void 0 ? void 0 : startNode.childNodes.item(min);
		if (isNodeOfType(node, "ELEMENT_NODE") && isElementOfType(node, "img")) return node;
	}
	return null;
}
function getProps(selection) {
	if (isSelection(selection)) return {
		startNode: selection.anchorNode,
		endNode: selection.focusNode,
		startOffset: selection.anchorOffset,
		endOffset: selection.focusOffset
	};
	else return {
		startNode: selection.startContainer,
		endNode: selection.endContainer,
		startOffset: selection.startOffset,
		endOffset: selection.endOffset
	};
}
function isSelection(selection) {
	return !!selection.getRangeAt;
}
var MouseLeftButton = 0;
var MouseRightButton$1 = 2;
var Up = "ArrowUp";
var Down = "ArrowDown";
var Left = "ArrowLeft";
var Right = "ArrowRight";
var Tab = "Tab";
var DEFAULT_SELECTION_BORDER_COLOR = "#DB626C";
var DEFAULT_TABLE_CELL_SELECTION_BACKGROUND_COLOR = "#C6C6C6";
var SelectionPlugin = function() {
	function SelectionPlugin$1(options) {
		var _this = this;
		var _a$5, _b$1;
		this.editor = null;
		this.disposer = null;
		this.logicalRootDisposer = null;
		this.isSafari = false;
		this.isMac = false;
		this.scrollTopCache = 0;
		this.onMouseMove = function(event$1) {
			var _a$6, _b$2, _c$1;
			if (_this.editor && _this.state.tableSelection) {
				var hasTableSelection = !!_this.state.tableSelection.lastCo;
				var currentNode = event$1.target;
				var domHelper = _this.editor.getDOMHelper();
				var range = _this.editor.getDocument().createRange();
				var startNode = _this.state.tableSelection.startNode;
				var isReverted = currentNode.compareDocumentPosition(startNode) == Node.DOCUMENT_POSITION_FOLLOWING;
				if (isReverted) {
					range.setStart(currentNode, 0);
					range.setEnd(startNode, isNodeOfType(startNode, "TEXT_NODE") ? (_b$2 = (_a$6 = startNode.nodeValue) === null || _a$6 === void 0 ? void 0 : _a$6.length) !== null && _b$2 !== void 0 ? _b$2 : 0 : startNode.childNodes.length);
				} else {
					range.setStart(startNode, 0);
					range.setEnd(currentNode, 0);
				}
				var tableStart = range.commonAncestorContainer;
				var newTableSelection = _this.parseTableSelection(tableStart, startNode, domHelper);
				if (newTableSelection) {
					var lastCo = findCoordinate(newTableSelection.parsedTable, currentNode, domHelper);
					if (newTableSelection.table != _this.state.tableSelection.table) {
						_this.state.tableSelection = newTableSelection;
						_this.state.tableSelection.lastCo = lastCo !== null && lastCo !== void 0 ? lastCo : void 0;
					}
					var updated = lastCo && _this.updateTableSelection(lastCo);
					if (hasTableSelection || updated) event$1.preventDefault();
				} else if (((_c$1 = _this.editor.getDOMSelection()) === null || _c$1 === void 0 ? void 0 : _c$1.type) == "table") _this.setDOMSelection({
					type: "range",
					range,
					isReverted
				}, _this.state.tableSelection);
			}
		};
		this.onDrop = function() {
			_this.detachMouseEvent();
		};
		this.getContainedTargetImage = function(event$1, previousSelection) {
			if (!_this.isMac || !previousSelection || previousSelection.type !== "image") return null;
			var target = event$1.target;
			if (isNodeOfType(target, "ELEMENT_NODE") && isElementOfType(target, "span") && target.firstChild === previousSelection.image) return previousSelection.image;
			return null;
		};
		this.onFocus = function() {
			var _a$6;
			if (!_this.state.skipReselectOnFocus && _this.state.selection) _this.setDOMSelection(_this.state.selection, _this.state.tableSelection);
			if (((_a$6 = _this.state.selection) === null || _a$6 === void 0 ? void 0 : _a$6.type) == "range" && !_this.isSafari) _this.state.selection = null;
			if (_this.scrollTopCache && _this.editor) {
				var sc = _this.editor.getScrollContainer();
				sc.scrollTop = _this.scrollTopCache;
				_this.scrollTopCache = 0;
			}
		};
		this.onBlur = function() {
			if (_this.editor) {
				if (!_this.state.selection) _this.state.selection = _this.editor.getDOMSelection();
				_this.scrollTopCache = _this.editor.getScrollContainer().scrollTop;
			}
		};
		this.onSelectionChange = function() {
			var _a$6;
			if (((_a$6 = _this.editor) === null || _a$6 === void 0 ? void 0 : _a$6.hasFocus()) && !_this.editor.isInShadowEdit()) {
				var newSelection = _this.editor.getDOMSelection();
				var selection = _this.editor.getDocument().getSelection();
				if (selection && selection.focusNode) {
					var image = isSingleImageInSelection(selection);
					if ((newSelection === null || newSelection === void 0 ? void 0 : newSelection.type) == "image" && !image) {
						var range = selection.getRangeAt(0);
						_this.editor.setDOMSelection({
							type: "range",
							range,
							isReverted: selection.focusNode != range.endContainer || selection.focusOffset != range.endOffset
						});
					} else if ((newSelection === null || newSelection === void 0 ? void 0 : newSelection.type) !== "image" && image) _this.editor.setDOMSelection({
						type: "image",
						image
					});
				}
				if ((newSelection === null || newSelection === void 0 ? void 0 : newSelection.type) == "range") {
					if (_this.isSafari) _this.state.selection = newSelection;
				}
			}
		};
		this.state = {
			selection: null,
			tableSelection: null,
			imageSelectionBorderColor: (_a$5 = options.imageSelectionBorderColor) !== null && _a$5 !== void 0 ? _a$5 : DEFAULT_SELECTION_BORDER_COLOR,
			imageSelectionBorderColorDark: options.imageSelectionBorderColor ? void 0 : DEFAULT_SELECTION_BORDER_COLOR,
			tableCellSelectionBackgroundColor: (_b$1 = options.tableCellSelectionBackgroundColor) !== null && _b$1 !== void 0 ? _b$1 : DEFAULT_TABLE_CELL_SELECTION_BACKGROUND_COLOR,
			tableCellSelectionBackgroundColorDark: options.tableCellSelectionBackgroundColor ? void 0 : DEFAULT_TABLE_CELL_SELECTION_BACKGROUND_COLOR
		};
	}
	SelectionPlugin$1.prototype.getName = function() {
		return "Selection";
	};
	SelectionPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		if (!this.state.imageSelectionBorderColorDark && this.state.imageSelectionBorderColor) this.state.imageSelectionBorderColorDark = editor.getColorManager().getDarkColor(this.state.imageSelectionBorderColor, void 0, "border");
		if (!this.state.tableCellSelectionBackgroundColorDark && this.state.tableCellSelectionBackgroundColor) this.state.tableCellSelectionBackgroundColorDark = editor.getColorManager().getDarkColor(this.state.tableCellSelectionBackgroundColor, void 0, "background");
		var env = this.editor.getEnvironment();
		var document$1 = this.editor.getDocument();
		this.isSafari = !!env.isSafari;
		this.isMac = !!env.isMac;
		document$1.addEventListener("selectionchange", this.onSelectionChange);
		if (this.isSafari) this.disposer = this.editor.attachDomEvent({
			focus: { beforeDispatch: this.onFocus },
			drop: { beforeDispatch: this.onDrop }
		});
		else this.disposer = this.editor.attachDomEvent({
			focus: { beforeDispatch: this.onFocus },
			blur: { beforeDispatch: this.onBlur },
			drop: { beforeDispatch: this.onDrop }
		});
	};
	SelectionPlugin$1.prototype.dispose = function() {
		var _a$5, _b$1;
		(_a$5 = this.editor) === null || _a$5 === void 0 || _a$5.getDocument().removeEventListener("selectionchange", this.onSelectionChange);
		if (this.disposer) {
			this.disposer();
			this.disposer = null;
		}
		(_b$1 = this.logicalRootDisposer) === null || _b$1 === void 0 || _b$1.call(this);
		this.logicalRootDisposer = null;
		this.detachMouseEvent();
		this.editor = null;
	};
	SelectionPlugin$1.prototype.getState = function() {
		return this.state;
	};
	SelectionPlugin$1.prototype.onPluginEvent = function(event$1) {
		var _a$5;
		if (!this.editor) return;
		switch (event$1.eventType) {
			case "mouseDown":
				this.onMouseDown(this.editor, event$1.rawEvent);
				break;
			case "mouseUp":
				this.onMouseUp();
				break;
			case "keyDown":
				this.onKeyDown(this.editor, event$1.rawEvent);
				break;
			case "contentChanged":
				this.state.tableSelection = null;
				break;
			case "scroll":
				if (!this.editor.hasFocus()) this.scrollTopCache = event$1.scrollContainer.scrollTop;
				break;
			case "logicalRootChanged":
				(_a$5 = this.logicalRootDisposer) === null || _a$5 === void 0 || _a$5.call(this);
				if (this.isSafari) this.logicalRootDisposer = this.editor.attachDomEvent({
					focus: { beforeDispatch: this.onFocus },
					drop: { beforeDispatch: this.onDrop }
				});
				else this.logicalRootDisposer = this.editor.attachDomEvent({
					focus: { beforeDispatch: this.onFocus },
					blur: { beforeDispatch: this.onBlur },
					drop: { beforeDispatch: this.onDrop }
				});
				break;
		}
	};
	SelectionPlugin$1.prototype.onMouseDown = function(editor, rawEvent) {
		var _a$5, _b$1, _c$1;
		var selection = editor.getDOMSelection();
		var image;
		if ((selection === null || selection === void 0 ? void 0 : selection.type) == "image" && (rawEvent.button == MouseLeftButton || rawEvent.button == MouseRightButton$1 && !this.getClickingImage(rawEvent) && !this.getContainedTargetImage(rawEvent, selection))) this.setDOMSelection(null, null);
		if ((image = (_a$5 = this.getClickingImage(rawEvent)) !== null && _a$5 !== void 0 ? _a$5 : this.getContainedTargetImage(rawEvent, selection)) && image.isContentEditable) {
			this.setDOMSelection({
				type: "image",
				image
			}, null);
			return;
		}
		if ((selection === null || selection === void 0 ? void 0 : selection.type) == "table" && rawEvent.button == MouseLeftButton) this.setDOMSelection(null, null);
		var tableSelection;
		var target = rawEvent.target;
		if (target && rawEvent.button == MouseLeftButton && (tableSelection = this.parseTableSelection(target, target, editor.getDOMHelper()))) {
			this.state.tableSelection = tableSelection;
			if (rawEvent.detail >= 3) {
				var lastCo = findCoordinate(tableSelection.parsedTable, rawEvent.target, editor.getDOMHelper());
				if (lastCo) {
					tableSelection.lastCo = lastCo;
					this.updateTableSelection(lastCo);
					rawEvent.preventDefault();
				}
			}
			(_c$1 = (_b$1 = this.state).mouseDisposer) === null || _c$1 === void 0 || _c$1.call(_b$1);
			this.state.mouseDisposer = editor.attachDomEvent({ mousemove: { beforeDispatch: this.onMouseMove } });
		}
	};
	SelectionPlugin$1.prototype.onMouseUp = function() {
		this.detachMouseEvent();
	};
	SelectionPlugin$1.prototype.onKeyDown = function(editor, rawEvent) {
		var _this = this;
		var _a$5, _b$1;
		var key = rawEvent.key;
		var selection = editor.getDOMSelection();
		var win = editor.getDocument().defaultView;
		switch (selection === null || selection === void 0 ? void 0 : selection.type) {
			case "image":
				if (!isModifierKey(rawEvent) && !rawEvent.shiftKey && selection.image.parentNode) {
					if (key === "Escape") {
						this.selectBeforeOrAfterElement(editor, selection.image);
						rawEvent.stopPropagation();
					} else if (key !== "Delete" && key !== "Backspace") this.selectBeforeOrAfterElement(editor, selection.image);
				}
				if ((isModifierKey(rawEvent) || rawEvent.shiftKey) && selection.image && !this.isSafari) {
					var range = selection.image.ownerDocument.createRange();
					range.selectNode(selection.image);
					this.setDOMSelection({
						type: "range",
						range,
						isReverted: false
					}, null);
				}
				break;
			case "range":
				if (key == Up || key == Down || key == Left || key == Right || key == Tab) {
					var start = selection.range.startContainer;
					this.state.tableSelection = this.parseTableSelection(start, start, editor.getDOMHelper());
					if (this.state.tableSelection && !rawEvent.defaultPrevented) if (key == Tab) {
						this.handleSelectionInTable(this.getTabKey(rawEvent));
						rawEvent.preventDefault();
					} else win === null || win === void 0 || win.requestAnimationFrame(function() {
						return _this.handleSelectionInTable(key);
					});
				}
				break;
			case "table":
				if (this.state.tableSelection == null) {
					var table = selection.table, firstRow = selection.firstRow, firstColumn = selection.firstColumn, lastRow = selection.lastRow, lastColumn = selection.lastColumn;
					var parsedTable = parseTableCells(table);
					if (parsedTable) {
						var firstCo = {
							row: firstRow,
							col: firstColumn
						};
						var lastCo = {
							row: lastRow,
							col: lastColumn
						};
						this.state.tableSelection = {
							table,
							parsedTable,
							firstCo,
							lastCo,
							startNode: ((_a$5 = findTableCellElement(parsedTable, firstCo)) === null || _a$5 === void 0 ? void 0 : _a$5.cell) || table
						};
					}
				}
				if ((_b$1 = this.state.tableSelection) === null || _b$1 === void 0 ? void 0 : _b$1.lastCo) {
					var shiftKey = rawEvent.shiftKey, key_1 = rawEvent.key;
					if (shiftKey && (key_1 == Left || key_1 == Right)) {
						var isRtl = (win === null || win === void 0 ? void 0 : win.getComputedStyle(this.state.tableSelection.table).direction) == "rtl";
						this.updateTableSelectionFromKeyboard(0, (key_1 == Left ? -1 : 1) * (isRtl ? -1 : 1));
						rawEvent.preventDefault();
					} else if (shiftKey && (key_1 == Up || key_1 == Down)) {
						this.updateTableSelectionFromKeyboard(key_1 == Up ? -1 : 1, 0);
						rawEvent.preventDefault();
					} else if (key_1 != "Shift" && !isCharacterValue(rawEvent)) {
						if (key_1 == Up || key_1 == Down || key_1 == Left || key_1 == Right) {
							this.setDOMSelection(null, this.state.tableSelection);
							win === null || win === void 0 || win.requestAnimationFrame(function() {
								return _this.handleSelectionInTable(key_1);
							});
						}
					}
				}
				break;
		}
	};
	SelectionPlugin$1.prototype.getTabKey = function(rawEvent) {
		return rawEvent.shiftKey ? "TabLeft" : "TabRight";
	};
	SelectionPlugin$1.prototype.handleSelectionInTable = function(key) {
		var _a$5, _b$1, _c$1, _d$1, _e$1, _f$1;
		if (!this.editor || !this.state.tableSelection) return;
		var selection = this.editor.getDOMSelection();
		var domHelper = this.editor.getDOMHelper();
		if ((selection === null || selection === void 0 ? void 0 : selection.type) == "range") {
			var _g = selection.range, collapsed = _g.collapsed, startContainer = _g.startContainer, endContainer = _g.endContainer, commonAncestorContainer = _g.commonAncestorContainer, isReverted = selection.isReverted;
			var start = isReverted ? endContainer : startContainer;
			var end = isReverted ? startContainer : endContainer;
			var tableSel = this.parseTableSelection(commonAncestorContainer, start, domHelper);
			if (!tableSel) return;
			var lastCo = findCoordinate(tableSel === null || tableSel === void 0 ? void 0 : tableSel.parsedTable, end, domHelper);
			var tabMove = false;
			var _h = this.state.tableSelection, parsedTable = _h.parsedTable, oldCo = _h.firstCo, table = _h.table;
			if (lastCo && tableSel.table == table) {
				if (lastCo.col != oldCo.col && (key == Up || key == Down)) {
					var change = key == Up ? -1 : 1;
					var originalTd = (_a$5 = findTableCellElement(parsedTable, oldCo)) === null || _a$5 === void 0 ? void 0 : _a$5.cell;
					var td = null;
					lastCo = {
						row: oldCo.row + change,
						col: oldCo.col
					};
					while (lastCo.row >= 0 && lastCo.row < parsedTable.length) {
						td = ((_b$1 = findTableCellElement(parsedTable, lastCo)) === null || _b$1 === void 0 ? void 0 : _b$1.cell) || null;
						if (td == originalTd) lastCo.row += change;
						else break;
					}
					if (collapsed && td) this.setRangeSelectionInTable(td, key == Up ? td.childNodes.length : 0, this.editor);
					else if (!td && (lastCo.row == -1 || lastCo.row <= parsedTable.length)) this.selectBeforeOrAfterElement(this.editor, table, change == 1, change != 1);
				} else if (key == "TabLeft" || key == "TabRight") {
					var reverse = key == "TabLeft";
					for (var step = reverse ? -1 : 1, row = (_c$1 = lastCo.row) !== null && _c$1 !== void 0 ? _c$1 : 0, col = ((_d$1 = lastCo.col) !== null && _d$1 !== void 0 ? _d$1 : 0) + step;; col += step) {
						if (col < 0 || col >= parsedTable[row].length) {
							row += step;
							if (row < 0) {
								this.selectBeforeOrAfterElement(this.editor, tableSel.table);
								break;
							} else if (row >= parsedTable.length) {
								this.selectBeforeOrAfterElement(this.editor, tableSel.table, true);
								break;
							}
							col = reverse ? parsedTable[row].length - 1 : 0;
						}
						var cell = parsedTable[row][col];
						if (typeof cell != "string") {
							tabMove = true;
							this.setRangeSelectionInTable(cell, 0, this.editor, true);
							lastCo.row = row;
							lastCo.col = col;
							break;
						}
					}
				} else this.state.tableSelection = null;
				if (collapsed && (lastCo.col != oldCo.col || lastCo.row != oldCo.row) && lastCo.row >= 0 && lastCo.row == parsedTable.length - 1 && lastCo.col == ((_e$1 = parsedTable[lastCo.row]) === null || _e$1 === void 0 ? void 0 : _e$1.length) - 1) (_f$1 = this.editor) === null || _f$1 === void 0 || _f$1.announce({ defaultStrings: "announceOnFocusLastCell" });
			}
			if (!collapsed && lastCo && !tabMove) {
				this.state.tableSelection = tableSel;
				this.updateTableSelection(lastCo);
			}
		}
	};
	SelectionPlugin$1.prototype.setRangeSelectionInTable = function(cell, nodeOffset, editor, selectAll) {
		var range = editor.getDocument().createRange();
		if (selectAll && cell.firstChild && cell.lastChild) {
			var cellStart = cell.firstChild;
			var cellEnd = cell.lastChild;
			var posStart = normalizePos(cellStart, 0);
			var posEnd = normalizePos(cellEnd, cellEnd.childNodes.length);
			range.setStart(posStart.node, posStart.offset);
			range.setEnd(posEnd.node, posEnd.offset);
			if (range.toString() === "") range.collapse(true);
		} else {
			var _a$5 = normalizePos(cell, nodeOffset), node = _a$5.node, offset = _a$5.offset;
			range.setStart(node, offset);
			range.collapse(true);
		}
		this.setDOMSelection({
			type: "range",
			range,
			isReverted: false
		}, null);
	};
	SelectionPlugin$1.prototype.updateTableSelectionFromKeyboard = function(rowChange, colChange) {
		var _a$5;
		if (((_a$5 = this.state.tableSelection) === null || _a$5 === void 0 ? void 0 : _a$5.lastCo) && this.editor) {
			var _b$1 = this.state.tableSelection, lastCo = _b$1.lastCo, parsedTable = _b$1.parsedTable;
			var row = lastCo.row + rowChange;
			var col = lastCo.col + colChange;
			if (row >= 0 && row < parsedTable.length && col >= 0 && col < parsedTable[row].length) this.updateTableSelection({
				row,
				col
			});
		}
	};
	SelectionPlugin$1.prototype.selectBeforeOrAfterElement = function(editor, element, after, setSelectionInNextSiblingElement) {
		var doc = editor.getDocument();
		var parent = element.parentNode;
		var index$1 = parent && toArray(parent.childNodes).indexOf(element);
		var sibling$1;
		if (parent && index$1 !== null && index$1 >= 0) {
			var range = doc.createRange();
			if (setSelectionInNextSiblingElement && (sibling$1 = after ? element.nextElementSibling : element.previousElementSibling) && isNodeOfType(sibling$1, "ELEMENT_NODE")) {
				range.selectNodeContents(sibling$1);
				range.collapse(false);
			} else {
				range.setStart(parent, index$1 + (after ? 1 : 0));
				range.collapse();
			}
			this.setDOMSelection({
				type: "range",
				range,
				isReverted: false
			}, null);
		}
	};
	SelectionPlugin$1.prototype.getClickingImage = function(event$1) {
		var target = event$1.target;
		return isNodeOfType(target, "ELEMENT_NODE") && isElementOfType(target, "img") ? target : null;
	};
	SelectionPlugin$1.prototype.parseTableSelection = function(tableStart, tdStart, domHelper) {
		var table;
		var parsedTable;
		var firstCo;
		if ((table = domHelper.findClosestElementAncestor(tableStart, "table")) && table.isContentEditable && (parsedTable = parseTableCells(table)) && (firstCo = findCoordinate(parsedTable, tdStart, domHelper))) return {
			table,
			parsedTable,
			firstCo,
			startNode: tdStart
		};
		else return null;
	};
	SelectionPlugin$1.prototype.updateTableSelection = function(lastCo) {
		if (this.state.tableSelection && this.editor) {
			var _a$5 = this.state.tableSelection, table = _a$5.table, firstCo = _a$5.firstCo, parsedTable = _a$5.parsedTable, startNode = _a$5.startNode;
			if (_a$5.lastCo || firstCo.row != lastCo.row || firstCo.col != lastCo.col) {
				this.state.tableSelection.lastCo = lastCo;
				this.setDOMSelection({
					type: "table",
					table,
					firstRow: firstCo.row,
					firstColumn: firstCo.col,
					lastRow: lastCo.row,
					lastColumn: lastCo.col,
					tableSelectionInfo: this.state.tableSelection
				}, {
					table,
					firstCo,
					lastCo,
					parsedTable,
					startNode
				});
				return true;
			}
		}
		return false;
	};
	SelectionPlugin$1.prototype.setDOMSelection = function(selection, tableSelection) {
		var _a$5;
		(_a$5 = this.editor) === null || _a$5 === void 0 || _a$5.setDOMSelection(selection);
		this.state.tableSelection = tableSelection;
	};
	SelectionPlugin$1.prototype.detachMouseEvent = function() {
		if (this.state.mouseDisposer) {
			this.state.mouseDisposer();
			this.state.mouseDisposer = void 0;
		}
	};
	return SelectionPlugin$1;
}();
function createSelectionPlugin(options) {
	return new SelectionPlugin(options);
}
var MAX_SIZE_LIMIT = 1e7;
var SnapshotsManagerImpl = function() {
	function SnapshotsManagerImpl$1(snapshots) {
		this.hasNewContentValue = false;
		this.snapshots = snapshots !== null && snapshots !== void 0 ? snapshots : {
			snapshots: [],
			totalSize: 0,
			currentIndex: -1,
			autoCompleteIndex: -1,
			maxSize: MAX_SIZE_LIMIT
		};
	}
	Object.defineProperty(SnapshotsManagerImpl$1.prototype, "hasNewContent", {
		get: function() {
			return this.hasNewContentValue;
		},
		set: function(value) {
			this.hasNewContentValue = value;
		},
		enumerable: false,
		configurable: true
	});
	SnapshotsManagerImpl$1.prototype.canMove = function(step) {
		var newIndex = this.snapshots.currentIndex + step;
		return newIndex >= 0 && newIndex < this.snapshots.snapshots.length;
	};
	SnapshotsManagerImpl$1.prototype.move = function(step) {
		var _a$5, _b$1;
		var result = null;
		if (this.canMove(step)) {
			this.snapshots.currentIndex += step;
			this.snapshots.autoCompleteIndex = -1;
			result = this.snapshots.snapshots[this.snapshots.currentIndex];
		}
		(_b$1 = (_a$5 = this.snapshots).onChanged) === null || _b$1 === void 0 || _b$1.call(_a$5, "move");
		return result;
	};
	SnapshotsManagerImpl$1.prototype.addSnapshot = function(snapshot$1, isAutoCompleteSnapshot) {
		var _a$5, _b$1;
		var currentSnapshot = this.snapshots.snapshots[this.snapshots.currentIndex];
		var isSameSnapshot = currentSnapshot && currentSnapshot.html == snapshot$1.html && !currentSnapshot.additionalState && !snapshot$1.additionalState && !currentSnapshot.entityStates && !snapshot$1.entityStates;
		var addSnapshot = !currentSnapshot || shouldAddSnapshot(currentSnapshot, snapshot$1);
		if (this.snapshots.currentIndex < 0 || addSnapshot) {
			this.clearRedo();
			this.snapshots.snapshots.push(snapshot$1);
			this.snapshots.currentIndex++;
			this.snapshots.totalSize += this.getSnapshotLength(snapshot$1);
			var removeCount = 0;
			while (removeCount < this.snapshots.snapshots.length && this.snapshots.totalSize > this.snapshots.maxSize) {
				this.snapshots.totalSize -= this.getSnapshotLength(this.snapshots.snapshots[removeCount]);
				removeCount++;
			}
			if (removeCount > 0) {
				this.snapshots.snapshots.splice(0, removeCount);
				this.snapshots.currentIndex -= removeCount;
				if (this.snapshots.autoCompleteIndex >= 0) this.snapshots.autoCompleteIndex -= removeCount;
			}
			if (isAutoCompleteSnapshot) this.snapshots.autoCompleteIndex = this.snapshots.currentIndex;
		} else if (isSameSnapshot) this.snapshots.snapshots.splice(this.snapshots.currentIndex, 1, snapshot$1);
		(_b$1 = (_a$5 = this.snapshots).onChanged) === null || _b$1 === void 0 || _b$1.call(_a$5, "add");
	};
	SnapshotsManagerImpl$1.prototype.clearRedo = function() {
		var _a$5, _b$1;
		if (this.canMove(1)) {
			var removedSize = 0;
			for (var i$1 = this.snapshots.currentIndex + 1; i$1 < this.snapshots.snapshots.length; i$1++) removedSize += this.getSnapshotLength(this.snapshots.snapshots[i$1]);
			this.snapshots.snapshots.splice(this.snapshots.currentIndex + 1);
			this.snapshots.totalSize -= removedSize;
			this.snapshots.autoCompleteIndex = -1;
			(_b$1 = (_a$5 = this.snapshots).onChanged) === null || _b$1 === void 0 || _b$1.call(_a$5, "clear");
		}
	};
	SnapshotsManagerImpl$1.prototype.canUndoAutoComplete = function() {
		return this.snapshots.autoCompleteIndex >= 0 && this.snapshots.currentIndex - this.snapshots.autoCompleteIndex == 1;
	};
	SnapshotsManagerImpl$1.prototype.getSnapshotLength = function(snapshot$1) {
		var _a$5, _b$1;
		return (_b$1 = (_a$5 = snapshot$1.html) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0;
	};
	return SnapshotsManagerImpl$1;
}();
function createSnapshotsManager(snapshots) {
	return new SnapshotsManagerImpl(snapshots);
}
function shouldAddSnapshot(currentSnapshot, snapshot$1) {
	return currentSnapshot.html !== snapshot$1.html || currentSnapshot.additionalState && snapshot$1.additionalState && JSON.stringify(currentSnapshot.additionalState) !== JSON.stringify(snapshot$1.additionalState) || !currentSnapshot.additionalState && snapshot$1.additionalState || currentSnapshot.entityStates && snapshot$1.entityStates && currentSnapshot.entityStates !== snapshot$1.entityStates || !currentSnapshot.entityStates && snapshot$1.entityStates;
}
function undo(editor) {
	editor.focus();
	var manager = editor.getSnapshotsManager();
	if (manager.hasNewContent) editor.takeSnapshot();
	var snapshot$1 = manager.move(-1);
	if (snapshot$1) editor.restoreSnapshot(snapshot$1);
}
var Backspace = "Backspace";
var Delete = "Delete";
var Enter = "Enter";
var UndoPlugin = function() {
	function UndoPlugin$1(options) {
		this.editor = null;
		this.state = {
			snapshotsManager: createSnapshotsManager(options.snapshots),
			isRestoring: false,
			isNested: false,
			autoCompleteInsertPoint: null,
			lastKeyPress: null
		};
	}
	UndoPlugin$1.prototype.getName = function() {
		return "Undo";
	};
	UndoPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
	};
	UndoPlugin$1.prototype.dispose = function() {
		this.editor = null;
	};
	UndoPlugin$1.prototype.getState = function() {
		return this.state;
	};
	UndoPlugin$1.prototype.willHandleEventExclusively = function(event$1) {
		return !!this.editor && event$1.eventType == "keyDown" && event$1.rawEvent.key == Backspace && !event$1.rawEvent.ctrlKey && this.canUndoAutoComplete(this.editor);
	};
	UndoPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (!this.editor) return;
		switch (event$1.eventType) {
			case "editorReady":
				var manager = this.state.snapshotsManager;
				var canUndo = manager.hasNewContent || manager.canMove(-1);
				var canRedo = manager.canMove(1);
				if (!canUndo && !canRedo) this.addUndoSnapshot();
				break;
			case "keyDown":
				this.onKeyDown(this.editor, event$1.rawEvent);
				break;
			case "keyPress":
				this.onKeyPress(this.editor, event$1.rawEvent);
				break;
			case "compositionEnd":
				this.clearRedoForInput();
				this.addUndoSnapshot();
				break;
			case "contentChanged":
				this.onContentChanged(event$1);
				break;
			case "beforeKeyboardEditing":
				this.onBeforeKeyboardEditing(event$1.rawEvent);
				break;
			case "mouseDown":
				if (this.state.snapshotsManager.hasNewContent) this.addUndoSnapshot();
				break;
		}
	};
	UndoPlugin$1.prototype.onKeyDown = function(editor, evt) {
		var snapshotsManager = this.state.snapshotsManager;
		if (evt.key == Backspace && !evt.altKey || evt.key == Delete) {
			if (evt.key == Backspace && !evt.ctrlKey && this.canUndoAutoComplete(editor)) {
				evt.preventDefault();
				undo(editor);
				this.state.autoCompleteInsertPoint = null;
				this.state.lastKeyPress = evt.key;
			} else if (!evt.defaultPrevented) {
				var selection = editor.getDOMSelection();
				if (selection && (selection.type != "range" || !selection.range.collapsed || this.state.lastKeyPress != evt.key || this.isCtrlOrMetaPressed(editor, evt))) this.addUndoSnapshot();
				snapshotsManager.hasNewContent = true;
				this.state.lastKeyPress = evt.key;
			}
		} else if (isCursorMovingKey(evt)) {
			if (snapshotsManager.hasNewContent) this.addUndoSnapshot();
			this.state.lastKeyPress = null;
		} else if (this.state.lastKeyPress == Backspace || this.state.lastKeyPress == Delete) {
			if (snapshotsManager.hasNewContent) this.addUndoSnapshot();
		}
	};
	UndoPlugin$1.prototype.onKeyPress = function(editor, evt) {
		if (evt.metaKey) return;
		var selection = editor.getDOMSelection();
		if (selection && (selection.type != "range" || !selection.range.collapsed) || evt.key == " " && this.state.lastKeyPress != " " || evt.key == Enter) {
			this.addUndoSnapshot();
			if (evt.key == Enter) this.state.snapshotsManager.hasNewContent = true;
		} else this.clearRedoForInput();
		this.state.lastKeyPress = evt.key;
	};
	UndoPlugin$1.prototype.onBeforeKeyboardEditing = function(event$1) {
		if (event$1.key != this.state.lastKeyPress) this.addUndoSnapshot();
		this.state.lastKeyPress = event$1.key;
		this.state.snapshotsManager.hasNewContent = true;
	};
	UndoPlugin$1.prototype.onContentChanged = function(event$1) {
		if (!this.state.isRestoring && !event$1.skipUndo) this.clearRedoForInput();
	};
	UndoPlugin$1.prototype.clearRedoForInput = function() {
		this.state.snapshotsManager.clearRedo();
		this.state.lastKeyPress = null;
		this.state.snapshotsManager.hasNewContent = true;
	};
	UndoPlugin$1.prototype.canUndoAutoComplete = function(editor) {
		var _a$5;
		var selection = editor.getDOMSelection();
		return this.state.snapshotsManager.canUndoAutoComplete() && (selection === null || selection === void 0 ? void 0 : selection.type) == "range" && selection.range.collapsed && selection.range.startContainer == ((_a$5 = this.state.autoCompleteInsertPoint) === null || _a$5 === void 0 ? void 0 : _a$5.node) && selection.range.startOffset == this.state.autoCompleteInsertPoint.offset;
	};
	UndoPlugin$1.prototype.addUndoSnapshot = function() {
		var _a$5;
		(_a$5 = this.editor) === null || _a$5 === void 0 || _a$5.takeSnapshot();
		this.state.autoCompleteInsertPoint = null;
	};
	UndoPlugin$1.prototype.isCtrlOrMetaPressed = function(editor, event$1) {
		return editor.getEnvironment().isMac ? event$1.metaKey : event$1.ctrlKey;
	};
	return UndoPlugin$1;
}();
function createUndoPlugin(option) {
	return new UndoPlugin(option);
}
function createEditorCorePlugins(options, contentDiv) {
	return {
		cache: createCachePlugin(options, contentDiv),
		format: createFormatPlugin(options),
		copyPaste: createCopyPastePlugin(options),
		domEvent: createDOMEventPlugin(options, contentDiv),
		lifecycle: createLifecyclePlugin(options, contentDiv),
		entity: createEntityPlugin(),
		selection: createSelectionPlugin(options),
		contextMenu: createContextMenuPlugin(options),
		undo: createUndoPlugin(options)
	};
}
function createEditorCore(contentDiv, options) {
	var _a$5, _b$1, _c$1;
	var corePlugins = createEditorCorePlugins(options, contentDiv);
	var domCreator = createDOMCreator(options.trustedHTMLHandler);
	return __assign(__assign({
		physicalRoot: contentDiv,
		logicalRoot: contentDiv,
		api: __assign(__assign({}, coreApiMap), options.coreApiOverride),
		originalApi: __assign({}, coreApiMap),
		plugins: __spreadArray(__spreadArray([
			corePlugins.cache,
			corePlugins.format,
			corePlugins.copyPaste,
			corePlugins.domEvent,
			corePlugins.selection,
			corePlugins.entity
		], __read(((_a$5 = options.plugins) !== null && _a$5 !== void 0 ? _a$5 : []).filter(function(x$1) {
			return !!x$1;
		})), false), [
			corePlugins.undo,
			corePlugins.contextMenu,
			corePlugins.lifecycle
		], false),
		environment: createEditorEnvironment(contentDiv, options),
		darkColorHandler: createDarkColorHandler(contentDiv, (_b$1 = options.getDarkColor) !== null && _b$1 !== void 0 ? _b$1 : getDarkColorFallback, options.knownColors, options.generateColorKey),
		trustedHTMLHandler: options.trustedHTMLHandler && !isDOMCreator(options.trustedHTMLHandler) ? options.trustedHTMLHandler : createTrustedHTMLHandler(domCreator),
		domCreator,
		domHelper: createDOMHelper(contentDiv, { cloneIndependentRoot: (_c$1 = options.experimentalFeatures) === null || _c$1 === void 0 ? void 0 : _c$1.includes("CloneIndependentRoot") })
	}, getPluginState(corePlugins)), {
		disposeErrorHandler: options.disposeErrorHandler,
		onFixUpModel: options.onFixUpModel,
		experimentalFeatures: options.experimentalFeatures ? __spreadArray([], __read(options.experimentalFeatures), false) : []
	});
}
function createEditorEnvironment(contentDiv, options) {
	var _a$5, _b$1, _c$1;
	var navigator = (_a$5 = contentDiv.ownerDocument.defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.navigator;
	var userAgent = (_b$1 = navigator === null || navigator === void 0 ? void 0 : navigator.userAgent) !== null && _b$1 !== void 0 ? _b$1 : "";
	var appVersion = (_c$1 = navigator === null || navigator === void 0 ? void 0 : navigator.appVersion) !== null && _c$1 !== void 0 ? _c$1 : "";
	return {
		domToModelSettings: createDomToModelSettings(options),
		modelToDomSettings: createModelToDomSettings(options),
		isMac: appVersion.indexOf("Mac") != -1,
		isAndroid: /android/i.test(userAgent),
		isIOS: /iPad|iPhone/.test(userAgent),
		isSafari: userAgent.indexOf("Safari") >= 0 && userAgent.indexOf("Chrome") < 0 && userAgent.indexOf("Android") < 0,
		isMobileOrTablet: getIsMobileOrTablet(userAgent)
	};
}
function getIsMobileOrTablet(userAgent) {
	if (/(android|bb\d+|meego).+mobile|avantgo|bada\/|blackberry|blazer|compal|elaine|fennec|hiptop|iemobile|ip(hone|od)|iris|kindle|lge |maemo|midp|mmp|mobile.+firefox|netfront|opera m(ob|in)i|palm( os)?|phone|p(ixi|re)\/|plucker|pocket|psp|series(4|6)0|symbian|treo|up\.(browser|link)|vodafone|wap|windows ce|xda|xiino|android|ipad|playbook|silk/i.test(userAgent) || /1207|6310|6590|3gso|4thp|50[1-6]i|770s|802s|a wa|abac|ac(er|oo|s\-)|ai(ko|rn)|al(av|ca|co)|amoi|an(ex|ny|yw)|aptu|ar(ch|go)|as(te|us)|attw|au(di|\-m|r |s )|avan|be(ck|ll|nq)|bi(lb|rd)|bl(ac|az)|br(e|v)w|bumb|bw\-(n|u)|c55\/|capi|ccwa|cdm\-|cell|chtm|cldc|cmd\-|co(mp|nd)|craw|da(it|ll|ng)|dbte|dc\-s|devi|dica|dmob|do(c|p)o|ds(12|\-d)|el(49|ai)|em(l2|ul)|er(ic|k0)|esl8|ez([4-7]0|os|wa|ze)|fetc|fly(\-|_)|g1 u|g560|gene|gf\-5|g\-mo|go(\.w|od)|gr(ad|un)|haie|hcit|hd\-(m|p|t)|hei\-|hi(pt|ta)|hp( i|ip)|hs\-c|ht(c(\-| |_|a|g|p|s|t)|tp)|hu(aw|tc)|i\-(20|go|ma)|i230|iac( |\-|\/)|ibro|idea|ig01|ikom|im1k|inno|ipaq|iris|ja(t|v)a|jbro|jemu|jigs|kddi|keji|kgt( |\/)|klon|kpt |kwc\-|kyo(c|k)|le(no|xi)|lg( g|\/(k|l|u)|50|54|\-[a-w])|libw|lynx|m1\-w|m3ga|m50\/|ma(te|ui|xo)|mc(01|21|ca)|m\-cr|me(rc|ri)|mi(o8|oa|ts)|mmef|mo(01|02|bi|de|do|t(\-| |o|v)|zz)|mt(50|p1|v )|mwbp|mywa|n10[0-2]|n20[2-3]|n30(0|2)|n50(0|2|5)|n7(0(0|1)|10)|ne((c|m)\-|on|tf|wf|wg|wt)|nok(6|i)|nzph|o2im|op(ti|wv)|oran|owg1|p800|pan(a|d|t)|pdxg|pg(13|\-([1-8]|c))|phil|pire|pl(ay|uc)|pn\-2|po(ck|rt|se)|prox|psio|pt\-g|qa\-a|qc(07|12|21|32|60|\-[2-7]|i\-)|qtek|r380|r600|raks|rim9|ro(ve|zo)|s55\/|sa(ge|ma|mm|ms|ny|va)|sc(01|h\-|oo|p\-)|sdk\/|se(c(\-|0|1)|47|mc|nd|ri)|sgh\-|shar|sie(\-|m)|sk\-0|sl(45|id)|sm(al|ar|b3|it|t5)|so(ft|ny)|sp(01|h\-|v\-|v )|sy(01|mb)|t2(18|50)|t6(00|10|18)|ta(gt|lk)|tcl\-|tdg\-|tel(i|m)|tim\-|t\-mo|to(pl|sh)|ts(70|m\-|m3|m5)|tx\-9|up(\.b|g1|si)|utst|v400|v750|veri|vi(rg|te)|vk(40|5[0-3]|\-v)|vm40|voda|vulc|vx(52|53|60|61|70|80|81|83|85|98)|w3c(\-| )|webc|whit|wi(g |nc|nw)|wmlb|wonu|x700|yas\-|your|zeto|zte\-/i.test(userAgent.substring(0, 4))) return true;
	return false;
}
function getPluginState(corePlugins) {
	return {
		domEvent: corePlugins.domEvent.getState(),
		copyPaste: corePlugins.copyPaste.getState(),
		cache: corePlugins.cache.getState(),
		format: corePlugins.format.getState(),
		lifecycle: corePlugins.lifecycle.getState(),
		entity: corePlugins.entity.getState(),
		selection: corePlugins.selection.getState(),
		contextMenu: corePlugins.contextMenu.getState(),
		undo: corePlugins.undo.getState()
	};
}
function getDarkColorFallback(color) {
	return color;
}
var Editor = function() {
	function Editor$1(contentDiv, options) {
		var _this = this;
		if (options === void 0) options = {};
		var _a$5;
		this.core = null;
		this.cloneOptionCallback = function(node, type) {
			if (type == "cache") return;
			var result = node.cloneNode(true);
			if (_this.isDarkMode()) {
				transformColor(result, true, "darkToLight", _this.getColorManager());
				result.style.color = result.style.color || "inherit";
				result.style.backgroundColor = result.style.backgroundColor || "inherit";
			}
			return result;
		};
		this.core = createEditorCore(contentDiv, options);
		var initialModel = (_a$5 = options.initialModel) !== null && _a$5 !== void 0 ? _a$5 : createEmptyModel(this.core.format.defaultFormat);
		this.core.api.setContentModel(this.core, initialModel, { ignoreSelection: true }, void 0, true);
		this.core.plugins.forEach(function(plugin) {
			return plugin.initialize(_this);
		});
	}
	Editor$1.prototype.dispose = function() {
		var _a$5;
		var core = this.getCore();
		for (var i$1 = core.plugins.length - 1; i$1 >= 0; i$1--) {
			var plugin = core.plugins[i$1];
			try {
				plugin.dispose();
			} catch (e$1) {
				(_a$5 = core.disposeErrorHandler) === null || _a$5 === void 0 || _a$5.call(core, plugin, e$1);
			}
		}
		core.darkColorHandler.reset();
		this.core = null;
	};
	Editor$1.prototype.isDisposed = function() {
		return !this.core;
	};
	Editor$1.prototype.getContentModelCopy = function(mode) {
		var core = this.getCore();
		switch (mode) {
			case "connected":
			case "disconnected": return cloneModel(core.api.createContentModel(core, { tryGetFromCache: false }), { includeCachedElement: this.cloneOptionCallback });
			case "clean":
				var domToModelContext = createDomToModelContextWithConfig(core.environment.domToModelSettings.calculated, core.api.createEditorContext(core, false));
				return cloneModel(domToContentModel(core.physicalRoot, domToModelContext), { includeCachedElement: this.cloneOptionCallback });
		}
	};
	Editor$1.prototype.getEnvironment = function() {
		return this.getCore().environment;
	};
	Editor$1.prototype.getDOMSelection = function() {
		var core = this.getCore();
		return core.api.getDOMSelection(core);
	};
	Editor$1.prototype.setDOMSelection = function(selection) {
		var core = this.getCore();
		core.api.setDOMSelection(core, selection);
	};
	Editor$1.prototype.setLogicalRoot = function(logicalRoot) {
		var core = this.getCore();
		core.api.setLogicalRoot(core, logicalRoot);
	};
	Editor$1.prototype.formatContentModel = function(formatter, options, domToModelOptions) {
		var core = this.getCore();
		core.api.formatContentModel(core, formatter, options, domToModelOptions);
	};
	Editor$1.prototype.getPendingFormat = function() {
		var _a$5, _b$1;
		return (_b$1 = (_a$5 = this.getCore().format.pendingFormat) === null || _a$5 === void 0 ? void 0 : _a$5.format) !== null && _b$1 !== void 0 ? _b$1 : null;
	};
	Editor$1.prototype.getDOMHelper = function() {
		return this.getCore().domHelper;
	};
	Editor$1.prototype.takeSnapshot = function(entityState) {
		var core = this.getCore();
		return core.api.addUndoSnapshot(core, false, entityState ? [entityState] : void 0);
	};
	Editor$1.prototype.restoreSnapshot = function(snapshot$1) {
		var core = this.getCore();
		core.api.restoreUndoSnapshot(core, snapshot$1);
	};
	Editor$1.prototype.getDocument = function() {
		return this.getCore().physicalRoot.ownerDocument;
	};
	Editor$1.prototype.focus = function() {
		var core = this.getCore();
		core.api.focus(core);
	};
	Editor$1.prototype.hasFocus = function() {
		return this.getCore().domHelper.hasFocus();
	};
	Editor$1.prototype.triggerEvent = function(eventType, data, broadcast) {
		if (broadcast === void 0) broadcast = false;
		var core = this.getCore();
		var event$1 = __assign({ eventType }, data);
		core.api.triggerEvent(core, event$1, broadcast);
		return event$1;
	};
	Editor$1.prototype.attachDomEvent = function(eventMap) {
		var core = this.getCore();
		return core.api.attachDomEvent(core, eventMap);
	};
	Editor$1.prototype.getSnapshotsManager = function() {
		return this.getCore().undo.snapshotsManager;
	};
	Editor$1.prototype.isDarkMode = function() {
		return this.getCore().lifecycle.isDarkMode;
	};
	Editor$1.prototype.setDarkModeState = function(isDarkMode) {
		var core = this.getCore();
		if (!!isDarkMode != core.lifecycle.isDarkMode) {
			transformColor(core.physicalRoot, false, isDarkMode ? "lightToDark" : "darkToLight", core.darkColorHandler);
			core.lifecycle.isDarkMode = !!isDarkMode;
			core.api.triggerEvent(core, {
				eventType: "contentChanged",
				source: isDarkMode ? ChangeSource.SwitchToDarkMode : ChangeSource.SwitchToLightMode,
				skipUndo: true
			}, true);
		}
	};
	Editor$1.prototype.isInShadowEdit = function() {
		return !!this.getCore().lifecycle.shadowEditFragment;
	};
	Editor$1.prototype.startShadowEdit = function() {
		var core = this.getCore();
		core.api.switchShadowEdit(core, true);
	};
	Editor$1.prototype.stopShadowEdit = function() {
		var core = this.getCore();
		core.api.switchShadowEdit(core, false);
	};
	Editor$1.prototype.getColorManager = function() {
		return this.getCore().darkColorHandler;
	};
	Editor$1.prototype.getTrustedHTMLHandler = function() {
		return this.getCore().trustedHTMLHandler;
	};
	Editor$1.prototype.getDOMCreator = function() {
		return this.getCore().domCreator;
	};
	Editor$1.prototype.getScrollContainer = function() {
		return this.getCore().domEvent.scrollContainer;
	};
	Editor$1.prototype.getVisibleViewport = function() {
		return this.getCore().api.getVisibleViewport(this.getCore());
	};
	Editor$1.prototype.setEditorStyle = function(key, cssRule, subSelectors) {
		var core = this.getCore();
		core.api.setEditorStyle(core, key, cssRule, subSelectors);
	};
	Editor$1.prototype.announce = function(announceData) {
		var core = this.getCore();
		core.api.announce(core, announceData);
	};
	Editor$1.prototype.isExperimentalFeatureEnabled = function(featureName) {
		return this.getCore().experimentalFeatures.indexOf(featureName) >= 0;
	};
	Editor$1.prototype.getCore = function() {
		if (!this.core) throw new Error("Editor is already disposed");
		return this.core;
	};
	return Editor$1;
}();
function redo(editor) {
	editor.focus();
	var snapshot$1 = editor.getSnapshotsManager().move(1);
	if (snapshot$1) editor.restoreSnapshot(snapshot$1);
}
function createElement(elementData, document$1) {
	if (!elementData || !elementData.tag) return null;
	var tag = elementData.tag, namespace = elementData.namespace, className = elementData.className, style = elementData.style, dataset = elementData.dataset, attributes = elementData.attributes, children = elementData.children;
	var result = namespace ? document$1.createElementNS(namespace, tag) : document$1.createElement(tag);
	if (style) result.setAttribute("style", style);
	if (className) result.className = className;
	if (dataset && isNodeOfType(result, "ELEMENT_NODE")) getObjectKeys(dataset).forEach(function(datasetName) {
		result.dataset[datasetName] = dataset[datasetName];
	});
	if (attributes) getObjectKeys(attributes).forEach(function(attrName) {
		result.setAttribute(attrName, attributes[attrName]);
	});
	if (children) children.forEach(function(child$1) {
		if (typeof child$1 === "string") result.appendChild(document$1.createTextNode(child$1));
		else if (child$1) {
			var childElement = createElement(child$1, document$1);
			if (childElement) result.appendChild(childElement);
		}
	});
	return result;
}
var MOUSE_EVENT_INFO_DESKTOP = (function() {
	return {
		MOUSEDOWN: "mousedown",
		MOUSEMOVE: "mousemove",
		MOUSEUP: "mouseup",
		getPageXY: getMouseEventPageXY
	};
})();
var MOUSE_EVENT_INFO_MOBILE = (function() {
	return {
		MOUSEDOWN: "touchstart",
		MOUSEMOVE: "touchmove",
		MOUSEUP: "touchend",
		getPageXY: getTouchEventPageXY
	};
})();
function getMouseEventPageXY(e$1) {
	return [e$1.pageX, e$1.pageY];
}
function getTouchEventPageXY(e$1) {
	var pageX = 0;
	var pageY = 0;
	if (e$1.targetTouches && e$1.targetTouches.length > 0) {
		var touch = e$1.targetTouches[0];
		pageX = touch.pageX;
		pageY = touch.pageY;
	}
	return [pageX, pageY];
}
var DragAndDropHelper = function() {
	function DragAndDropHelper$1(trigger, context, onSubmit, handler, zoomScale, forceMobile) {
		var _this = this;
		this.trigger = trigger;
		this.context = context;
		this.onSubmit = onSubmit;
		this.handler = handler;
		this.zoomScale = zoomScale;
		this.initX = 0;
		this.initY = 0;
		this.initValue = void 0;
		this.onMouseDown = function(e$1) {
			var _a$5;
			var _b$1, _c$1;
			e$1.preventDefault();
			e$1.stopPropagation();
			_this.addDocumentEvents();
			_a$5 = __read(_this.dndMouse.getPageXY(e$1), 2), _this.initX = _a$5[0], _this.initY = _a$5[1];
			_this.initValue = (_c$1 = (_b$1 = _this.handler).onDragStart) === null || _c$1 === void 0 ? void 0 : _c$1.call(_b$1, _this.context, e$1);
		};
		this.onMouseMove = function(e$1) {
			var _a$5, _b$1, _c$1;
			e$1.preventDefault();
			var _d$1 = __read(_this.dndMouse.getPageXY(e$1), 2), pageX = _d$1[0], pageY = _d$1[1];
			var deltaX = (pageX - _this.initX) / _this.zoomScale;
			var deltaY = (pageY - _this.initY) / _this.zoomScale;
			if (_this.initValue && ((_b$1 = (_a$5 = _this.handler).onDragging) === null || _b$1 === void 0 ? void 0 : _b$1.call(_a$5, _this.context, e$1, _this.initValue, deltaX, deltaY))) (_c$1 = _this.onSubmit) === null || _c$1 === void 0 || _c$1.call(_this, _this.context, _this.trigger);
		};
		this.onMouseUp = function(e$1) {
			var _a$5, _b$1, _c$1;
			e$1.preventDefault();
			_this.removeDocumentEvents();
			if ((_b$1 = (_a$5 = _this.handler).onDragEnd) === null || _b$1 === void 0 ? void 0 : _b$1.call(_a$5, _this.context, e$1, _this.initValue)) (_c$1 = _this.onSubmit) === null || _c$1 === void 0 || _c$1.call(_this, _this.context, _this.trigger);
		};
		this.dndMouse = forceMobile ? MOUSE_EVENT_INFO_MOBILE : MOUSE_EVENT_INFO_DESKTOP;
		trigger.addEventListener(this.dndMouse.MOUSEDOWN, this.onMouseDown);
	}
	DragAndDropHelper$1.prototype.dispose = function() {
		this.trigger.removeEventListener(this.dndMouse.MOUSEDOWN, this.onMouseDown);
		this.removeDocumentEvents();
	};
	Object.defineProperty(DragAndDropHelper$1.prototype, "mouseType", {
		get: function() {
			return this.dndMouse == MOUSE_EVENT_INFO_MOBILE ? "touch" : "mouse";
		},
		enumerable: false,
		configurable: true
	});
	DragAndDropHelper$1.prototype.addDocumentEvents = function() {
		var doc = this.trigger.ownerDocument;
		doc.addEventListener(this.dndMouse.MOUSEMOVE, this.onMouseMove, true);
		doc.addEventListener(this.dndMouse.MOUSEUP, this.onMouseUp, true);
	};
	DragAndDropHelper$1.prototype.removeDocumentEvents = function() {
		var doc = this.trigger.ownerDocument;
		doc.removeEventListener(this.dndMouse.MOUSEMOVE, this.onMouseMove, true);
		doc.removeEventListener(this.dndMouse.MOUSEUP, this.onMouseUp, true);
	};
	return DragAndDropHelper$1;
}();
function getCMTableFromTable(editor, table) {
	var model = createContentModelDocument();
	var context = createDomToModelContext({
		zoomScale: editor.getDOMHelper().calculateZoomScale(),
		recalculateTableSize: true
	});
	context.elementProcessors.element(model, table, context);
	var firstBlock = model.blocks[0];
	return (firstBlock === null || firstBlock === void 0 ? void 0 : firstBlock.blockType) == "Table" ? firstBlock : void 0;
}
var CELL_RESIZER_WIDTH = 4;
var HORIZONTAL_RESIZER_ID = "horizontalResizer";
var VERTICAL_RESIZER_ID = "verticalResizer";
function createCellResizer(editor, td, table, isRTL, isHorizontal, onStart, onEnd, anchorContainer, onTableEditorCreated) {
	var document$1 = td.ownerDocument;
	var createElementData = {
		tag: "div",
		style: "position: fixed; cursor: " + (isHorizontal ? "row" : "col") + "-resize; user-select: none"
	};
	var zoomScale = editor.getDOMHelper().calculateZoomScale();
	var div = createElement(createElementData, document$1);
	(anchorContainer || document$1.body).appendChild(div);
	var context = {
		editor,
		td,
		table,
		isRTL,
		zoomScale,
		onStart,
		originalWidth: parseValueWithUnit(table.style.width)
	};
	var setPosition = isHorizontal ? setHorizontalPosition : setVerticalPosition;
	setPosition(context, div);
	return {
		node: td,
		div,
		featureHandler: new CellResizer(div, context, setPosition, {
			onDragStart: onDragStart$2,
			onDragging: isHorizontal ? onDraggingHorizontal : onDraggingVertical,
			onDragEnd: onEnd
		}, zoomScale, editor.getEnvironment().isMobileOrTablet, onTableEditorCreated)
	};
}
var CellResizer = function(_super) {
	__extends(CellResizer$1, _super);
	function CellResizer$1(trigger, context, onSubmit, handler, zoomScale, forceMobile, onTableEditorCreated) {
		var _this = _super.call(this, trigger, context, onSubmit, handler, zoomScale, forceMobile) || this;
		_this.disposer = onTableEditorCreated === null || onTableEditorCreated === void 0 ? void 0 : onTableEditorCreated("CellResizer", trigger);
		return _this;
	}
	CellResizer$1.prototype.dispose = function() {
		var _a$5;
		(_a$5 = this.disposer) === null || _a$5 === void 0 || _a$5.call(this);
		this.disposer = void 0;
		_super.prototype.dispose.call(this);
	};
	return CellResizer$1;
}(DragAndDropHelper);
function onDragStart$2(context, event$1) {
	var td = context.td, onStart = context.onStart;
	var rect = normalizeRect(td.getBoundingClientRect());
	var columnIndex = td.cellIndex;
	var row = td.parentElement && isElementOfType(td.parentElement, "tr") ? td.parentElement : void 0;
	var rowIndex = row === null || row === void 0 ? void 0 : row.rowIndex;
	if (rowIndex == void 0) return {
		cmTable: void 0,
		anchorColumn: void 0,
		anchorRow: void 0,
		anchorRowHeight: -1,
		allWidths: []
	};
	var editor = context.editor, table = context.table;
	var cmTable = getCMTableFromTable(editor, table);
	if (rect && cmTable) {
		onStart();
		return {
			cmTable,
			anchorColumn: columnIndex,
			anchorRow: rowIndex,
			anchorRowHeight: cmTable.rows[rowIndex].height,
			allWidths: __spreadArray([], __read(cmTable.widths), false)
		};
	} else return {
		cmTable,
		anchorColumn: void 0,
		anchorRow: void 0,
		anchorRowHeight: -1,
		allWidths: []
	};
}
function onDraggingHorizontal(context, event$1, initValue, deltaX, deltaY) {
	var table = context.table;
	var cmTable = initValue.cmTable, anchorRow = initValue.anchorRow, anchorRowHeight = initValue.anchorRowHeight;
	if (cmTable && anchorRow != void 0 && cmTable.rows[anchorRow] != void 0) {
		mutateBlock(cmTable).rows[anchorRow].height = (anchorRowHeight !== null && anchorRowHeight !== void 0 ? anchorRowHeight : 0) + deltaY;
		var newHeight = Math.max(cmTable.rows[anchorRow].height, 22);
		var tableRow = table.rows[anchorRow];
		for (var col = 0; col < tableRow.cells.length; col++) {
			var td = tableRow.cells[col];
			td.style.height = newHeight + "px";
			td.style.boxSizing = "border-box";
		}
		return true;
	} else return false;
}
function onDraggingVertical(context, event$1, initValue, deltaX) {
	var table = context.table, isRTL = context.isRTL;
	var cmTable = initValue.cmTable, anchorColumn = initValue.anchorColumn, allWidths = initValue.allWidths;
	if (cmTable && anchorColumn != void 0) {
		var mutableTable = mutateBlock(cmTable);
		var lastColumn = anchorColumn == cmTable.widths.length - 1;
		var change = deltaX * (isRTL ? -1 : 1);
		if (lastColumn) {
			var newWidth = Math.max(allWidths[anchorColumn] + change, 30);
			mutableTable.widths[anchorColumn] = newWidth;
		} else {
			var anchorChange = allWidths[anchorColumn] + change;
			var nextAnchorChange = allWidths[anchorColumn + 1] - change;
			if (anchorChange < 30 || nextAnchorChange < 30) return false;
			mutableTable.widths[anchorColumn] = anchorChange;
			mutableTable.widths[anchorColumn + 1] = nextAnchorChange;
		}
		for (var row = 0; row < table.rows.length; row++) {
			var tableRow = table.rows[row];
			for (var col = 0; col < tableRow.cells.length; col++) {
				var td = tableRow.cells[col];
				td.style.width = cmTable.widths[col] + "px";
				td.style.boxSizing = "border-box";
			}
		}
		if (context.originalWidth > 0) {
			var newWidth = context.originalWidth + change + "px";
			mutableTable.format.width = newWidth;
			table.style.width = newWidth;
		}
		return true;
	} else return false;
}
function setHorizontalPosition(context, trigger) {
	var td = context.td;
	var rect = normalizeRect(td.getBoundingClientRect());
	if (rect) {
		trigger.id = HORIZONTAL_RESIZER_ID;
		trigger.style.top = rect.bottom - CELL_RESIZER_WIDTH + "px";
		trigger.style.left = rect.left + "px";
		trigger.style.width = rect.right - rect.left + "px";
		trigger.style.height = CELL_RESIZER_WIDTH + "px";
	}
}
function setVerticalPosition(context, trigger) {
	var td = context.td, isRTL = context.isRTL;
	var rect = normalizeRect(td.getBoundingClientRect());
	if (rect) {
		trigger.id = VERTICAL_RESIZER_ID;
		trigger.style.top = rect.top + "px";
		trigger.style.left = (isRTL ? rect.left : rect.right) - CELL_RESIZER_WIDTH + 1 + "px";
		trigger.style.width = CELL_RESIZER_WIDTH + "px";
		trigger.style.height = rect.bottom - rect.top + "px";
	}
}
function getIntersectedRect(elements, additionalRects) {
	if (additionalRects === void 0) additionalRects = [];
	var rects = elements.map(function(element) {
		return normalizeRect(element.getBoundingClientRect());
	}).concat(additionalRects).filter(function(element) {
		return !!element;
	});
	var result = {
		top: Math.max.apply(Math, __spreadArray([], __read(rects.map(function(r$1) {
			return r$1.top;
		})), false)),
		bottom: Math.min.apply(Math, __spreadArray([], __read(rects.map(function(r$1) {
			return r$1.bottom;
		})), false)),
		left: Math.max.apply(Math, __spreadArray([], __read(rects.map(function(r$1) {
			return r$1.left;
		})), false)),
		right: Math.min.apply(Math, __spreadArray([], __read(rects.map(function(r$1) {
			return r$1.right;
		})), false))
	};
	return result.top < result.bottom && result.left < result.right ? result : null;
}
var EN_SPACE = " ";
var REGULAR_SPACE = " ";
var NON_BREAK_SPACES = "\xA0";
function countTabsSpaces(text) {
	var spaces = countSpacesBeforeText(text);
	return Math.floor(spaces / 4);
}
function countSpacesBeforeText(str) {
	var e_1, _a$5;
	var count = 0;
	try {
		for (var str_1 = __values(str), str_1_1 = str_1.next(); !str_1_1.done; str_1_1 = str_1.next()) {
			var char = str_1_1.value;
			if (char === EN_SPACE || char === NON_BREAK_SPACES || char == REGULAR_SPACE) count++;
			else break;
		}
	} catch (e_1_1) {
		e_1 = { error: e_1_1 };
	} finally {
		try {
			if (str_1_1 && !str_1_1.done && (_a$5 = str_1.return)) _a$5.call(str_1);
		} finally {
			if (e_1) throw e_1.error;
		}
	}
	return count;
}
function adjustListIndentation(listItem) {
	var block = listItem.blocks[0];
	if (block.blockType == "Paragraph" && block.segments.length > 0 && block.segments[0].segmentType == "Text") {
		var tabSpaces_1 = countTabsSpaces(block.segments[0].text);
		if (tabSpaces_1 > 0) {
			mutateSegment(block, block.segments[0], function(textSegment) {
				textSegment.text = textSegment.text.substring(tabSpaces_1 * 4);
			});
			listItem.levels[0].format.marginLeft = tabSpaces_1 * 40 + "px";
		}
	}
}
function adjustTableIndentation(insertPoint, table) {
	var paragraph = insertPoint.paragraph, marker = insertPoint.marker;
	var indentationMargin = getTableIndentation(paragraph);
	if (indentationMargin) {
		insertPoint.paragraph.segments = [marker];
		if (insertPoint.paragraph.format.direction == "rtl") table.format.marginRight = indentationMargin * 40 + "px";
		else table.format.marginLeft = indentationMargin * 40 + "px";
	}
}
var getTableIndentation = function(paragraph) {
	var e_2, _a$5;
	var tabsNumber = 0;
	var segments = paragraph.segments;
	if (!paragraph.segments.every(function(s$1) {
		return s$1.segmentType == "Text" && s$1.text.trim().length == 0 || s$1.segmentType == "SelectionMarker" || s$1.segmentType == "Br";
	})) return;
	var numberOfSegments = 0;
	try {
		for (var segments_1 = __values(segments), segments_1_1 = segments_1.next(); !segments_1_1.done; segments_1_1 = segments_1.next()) {
			var seg = segments_1_1.value;
			if (seg.segmentType === "Text") {
				tabsNumber = tabsNumber + countTabsSpaces(seg.text);
				numberOfSegments++;
			} else break;
		}
	} catch (e_2_1) {
		e_2 = { error: e_2_1 };
	} finally {
		try {
			if (segments_1_1 && !segments_1_1.done && (_a$5 = segments_1.return)) _a$5.call(segments_1);
		} finally {
			if (e_2) throw e_2.error;
		}
	}
	if (segments.length - 2 <= numberOfSegments) return tabsNumber;
};
function createTableStructure(parent, columns, rows) {
	var table = createTable(rows);
	addBlock(parent, table);
	table.rows.forEach(function(row) {
		for (var i$1 = 0; i$1 < columns; i$1++) {
			var cell = createTableCell();
			row.cells.push(cell);
		}
	});
	return table;
}
function insertTable(editor, columns, rows, format) {
	editor.focus();
	editor.formatContentModel(function(model, context) {
		var _a$5, _b$1, _c$1;
		var insertPosition = deleteSelection(model, [], context).insertPoint;
		if (insertPosition) {
			var doc = createContentModelDocument();
			var table = createTableStructure(doc, columns, rows);
			normalizeTable$1(table, editor.getPendingFormat() || insertPosition.marker.format);
			adjustTableIndentation(insertPosition, table);
			format = format || { verticalAlign: "top" };
			applyTableFormat(table, format);
			mergeModel(model, doc, context, {
				insertPosition,
				mergeFormat: "mergeAll"
			});
			var firstBlock = (_b$1 = (_a$5 = table.rows[0]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[0]) === null || _b$1 === void 0 ? void 0 : _b$1.blocks[0];
			if ((firstBlock === null || firstBlock === void 0 ? void 0 : firstBlock.blockType) == "Paragraph") {
				var marker = createSelectionMarker((_c$1 = firstBlock.segments[0]) === null || _c$1 === void 0 ? void 0 : _c$1.format);
				firstBlock.segments.unshift(marker);
				setSelection(model, marker);
			}
			return true;
		} else return false;
	}, { apiName: "insertTable" });
}
function alignTable(table, operation) {
	table.format.marginLeft = operation == "alignLeft" ? "" : "auto";
	table.format.marginRight = operation == "alignRight" ? "" : "auto";
}
function deleteTable(table) {
	table.rows = [];
}
function collapseTableSelection(rows, selection) {
	var _a$5;
	var firstColumn = selection.firstColumn;
	var cell = (_a$5 = rows[selection.firstRow]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[firstColumn];
	if (cell) addSegment(mutateBlock(cell), createSelectionMarker());
}
function deleteTableColumn(table) {
	var sel = getSelectedCells(table);
	if (sel) {
		for (var rowIndex = 0; rowIndex < table.rows.length; rowIndex++) {
			var cellInNextCol = table.rows[rowIndex].cells[sel.lastColumn + 1];
			if (cellInNextCol) mutateBlock(cellInNextCol).spanLeft = cellInNextCol.spanLeft && table.rows[rowIndex].cells[sel.firstColumn].spanLeft;
			table.rows[rowIndex].cells.splice(sel.firstColumn, sel.lastColumn - sel.firstColumn + 1);
		}
		table.widths.splice(sel.firstColumn, sel.lastColumn - sel.firstColumn + 1);
		collapseTableSelection(table.rows, sel);
	}
}
function deleteTableRow(table) {
	var sel = getSelectedCells(table);
	if (sel) {
		table.rows[sel.firstRow].cells.forEach(function(cell, colIndex) {
			var _a$5;
			var cellInNextCell = (_a$5 = table.rows[sel.lastRow + 1]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[colIndex];
			if (cellInNextCell) mutateBlock(cellInNextCell).spanAbove = cellInNextCell.spanAbove && cell.spanAbove;
		});
		table.rows.splice(sel.firstRow, sel.lastRow - sel.firstRow + 1);
		collapseTableSelection(table.rows, sel);
	}
}
function ensureFocusableParagraphForTable(model, path, table) {
	var _a$5, _b$1;
	var paragraph;
	var firstCell = (_a$5 = table.rows.filter(function(row) {
		return row.cells.length > 0;
	})[0]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[0];
	if (firstCell) {
		paragraph = firstCell.blocks.filter(function(block$1) {
			return block$1.blockType == "Paragraph";
		})[0];
		if (!paragraph) {
			paragraph = createEmptyParagraph(model);
			mutateBlock(firstCell).blocks.push(paragraph);
		}
	} else {
		var block = table;
		var parent_1;
		paragraph = createEmptyParagraph(model);
		while (parent_1 = path.shift()) {
			var index$1 = (_b$1 = parent_1.blocks.indexOf(block)) !== null && _b$1 !== void 0 ? _b$1 : -1;
			if (parent_1 && index$1 >= 0) mutateBlock(parent_1).blocks.splice(index$1, 1, paragraph);
			if (parent_1.blockGroupType == "FormatContainer" && parent_1.blocks.length == 1 && parent_1.blocks[0] == paragraph) block = parent_1;
			else break;
		}
	}
	return paragraph;
}
function createEmptyParagraph(model) {
	var newPara = createParagraph(false, void 0, model.format);
	var br = createBr(model.format);
	newPara.segments.push(br);
	return newPara;
}
function formatTableWithContentModel(editor, apiName, callback, selectionOverride) {
	editor.focus();
	editor.formatContentModel(function(model) {
		var _a$5 = __read(getFirstSelectedTable(model), 2), readonlyTableModel = _a$5[0], path = _a$5[1];
		if (readonlyTableModel) {
			var tableModel = mutateBlock(readonlyTableModel);
			callback(tableModel);
			if (!hasSelectionInBlock(tableModel)) {
				var paragraph = ensureFocusableParagraphForTable(model, path, tableModel);
				if (paragraph) {
					var marker = createSelectionMarker(model.format);
					paragraph.segments.unshift(marker);
					setParagraphNotImplicit(paragraph);
					setSelection(model, marker);
				}
			}
			normalizeTable$1(tableModel, model.format);
			if (hasMetadata(tableModel)) applyTableFormat(tableModel, void 0, true);
			return true;
		} else return false;
	}, {
		apiName,
		selectionOverride
	}, { recalculateTableSize: "selected" });
}
function clearSelectedCells(table, sel) {
	if (sel.firstColumn >= 0 && sel.firstRow >= 0 && sel.lastRow < table.rows.length && sel.lastColumn < table.rows[sel.lastRow].cells.length) for (var i$1 = sel.firstRow; i$1 <= sel.lastRow; i$1++) {
		var row = table.rows[i$1];
		for (var j$1 = sel.firstColumn; j$1 <= sel.lastColumn; j$1++) {
			var cell = row.cells[j$1];
			if (cell) {
				if (cell.isSelected) mutateBlock(cell).isSelected = false;
				setSelection(cell);
			}
		}
	}
}
function insertTableColumn(table, operation) {
	var sel = getSelectedCells(table);
	var insertLeft = operation == "insertLeft";
	if (sel) {
		clearSelectedCells(table, sel);
		for (var i$1 = sel === null || sel === void 0 ? void 0 : sel.firstColumn; i$1 <= sel.lastColumn; i$1++) {
			table.rows.forEach(function(row) {
				var cell = row.cells[insertLeft ? sel.firstColumn : sel.lastColumn];
				var newCell = createTableCell(cell.spanLeft, cell.spanAbove, cell.isHeader, cell.format, cell.dataset);
				newCell.isSelected = true;
				row.cells.splice(insertLeft ? sel.firstColumn : sel.lastColumn + 1, 0, newCell);
			});
			table.widths.splice(insertLeft ? sel.firstColumn : sel.lastColumn + 1, 0, table.widths[insertLeft ? sel.firstColumn : sel.lastColumn]);
		}
	}
}
function insertTableRow(table, operation) {
	var sel = getSelectedCells(table);
	var insertAbove = operation == "insertAbove";
	if (sel) {
		clearSelectedCells(table, sel);
		for (var i$1 = sel.firstRow; i$1 <= sel.lastRow; i$1++) {
			var sourceRow = table.rows[insertAbove ? sel.firstRow : sel.lastRow];
			table.rows.splice(insertAbove ? sel.firstRow : sel.lastRow + 1, 0, {
				format: __assign({}, sourceRow.format),
				cells: sourceRow.cells.map(function(cell) {
					var newCell = createTableCell(cell.spanLeft, cell.spanAbove, cell.isHeader, cell.format, cell.dataset);
					newCell.isSelected = true;
					return newCell;
				}),
				height: sourceRow.height
			});
		}
	}
}
function canMergeCells(rows, firstRow, firstCol, lastRow, lastCol) {
	var noSpanAbove = firstCol == lastCol || rows[firstRow].cells.every(function(cell, colIndex) {
		return colIndex < firstCol || colIndex > lastCol || !cell.spanAbove;
	});
	var noSpanLeft = firstRow == lastRow || rows.every(function(row, rowIndex) {
		return rowIndex < firstRow || rowIndex > lastRow || !row.cells[firstCol].spanLeft;
	});
	var noDifferentBelowSpan = rows[lastRow].cells.map(function(_, colIndex) {
		return colIndex >= firstCol && colIndex <= lastCol ? getBelowSpanCount(rows, lastRow, colIndex) : -1;
	}).every(function(x$1, _, a$1) {
		return x$1 < 0 || x$1 == a$1[firstCol];
	});
	var noDifferentRightSpan = rows.map(function(_, rowIndex) {
		return rowIndex >= firstRow && rowIndex <= lastRow ? getRightSpanCount(rows, rowIndex, lastCol) : -1;
	}).every(function(x$1, _, a$1) {
		return x$1 < 0 || x$1 == a$1[firstRow];
	});
	return noSpanAbove && noSpanLeft && noDifferentBelowSpan && noDifferentRightSpan;
}
function getBelowSpanCount(rows, rowIndex, colIndex) {
	var _a$5, _b$1;
	var spanCount = 0;
	for (var row = rowIndex + 1; row < rows.length; row++) if ((_b$1 = (_a$5 = rows[row]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[colIndex]) === null || _b$1 === void 0 ? void 0 : _b$1.spanAbove) spanCount++;
	else break;
	return spanCount;
}
function getRightSpanCount(rows, rowIndex, colIndex) {
	var _a$5, _b$1, _c$1;
	var spanCount = 0;
	for (var col = colIndex + 1; col < ((_a$5 = rows[rowIndex]) === null || _a$5 === void 0 ? void 0 : _a$5.cells.length); col++) if ((_c$1 = (_b$1 = rows[rowIndex]) === null || _b$1 === void 0 ? void 0 : _b$1.cells[col]) === null || _c$1 === void 0 ? void 0 : _c$1.spanLeft) spanCount++;
	else break;
	return spanCount;
}
function mergeTableCells(table) {
	var sel = getSelectedCells(table);
	if (sel && canMergeCells(table.rows, sel.firstRow, sel.firstColumn, sel.lastRow, sel.lastColumn)) for (var rowIndex = sel.firstRow; rowIndex <= sel.lastRow; rowIndex++) for (var colIndex = sel.firstColumn; colIndex <= sel.lastColumn; colIndex++) {
		var cell = table.rows[rowIndex].cells[colIndex];
		if (cell) {
			var mutableCell = mutateBlock(cell);
			mutableCell.spanLeft = colIndex > sel.firstColumn;
			mutableCell.spanAbove = rowIndex > sel.firstRow;
		}
	}
}
function mergeTableColumn(table, operation) {
	var _a$5, _b$1, _c$1, _d$1;
	var sel = getSelectedCells(table);
	var mergeLeft = operation == "mergeLeft";
	if (sel) {
		var mergingColIndex = mergeLeft ? sel.firstColumn : sel.lastColumn + 1;
		if (mergingColIndex > 0 && mergingColIndex < table.rows[0].cells.length) for (var rowIndex = sel.firstRow; rowIndex <= sel.lastRow; rowIndex++) {
			var cell = (_a$5 = table.rows[rowIndex]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[mergingColIndex];
			if (cell && canMergeCells(table.rows, rowIndex, mergingColIndex - 1, rowIndex, mergingColIndex)) {
				mutateBlock(cell).spanLeft = true;
				var newSelectedCol = mergingColIndex;
				while ((_c$1 = (_b$1 = table.rows[rowIndex]) === null || _b$1 === void 0 ? void 0 : _b$1.cells[newSelectedCol]) === null || _c$1 === void 0 ? void 0 : _c$1.spanLeft) {
					mutateBlock(table.rows[rowIndex].cells[newSelectedCol]);
					newSelectedCol--;
				}
				var newCell = (_d$1 = table.rows[rowIndex]) === null || _d$1 === void 0 ? void 0 : _d$1.cells[newSelectedCol];
				if (newCell) mutateBlock(newCell).isSelected = true;
			}
		}
	}
}
function mergeTableRow(table, operation) {
	var _a$5, _b$1, _c$1;
	var sel = getSelectedCells(table);
	var mergeAbove = operation == "mergeAbove";
	if (sel) {
		var mergingRowIndex = mergeAbove ? sel.firstRow : sel.lastRow + 1;
		if (mergingRowIndex > 0 && mergingRowIndex < table.rows.length) for (var colIndex = sel.firstColumn; colIndex <= sel.lastColumn; colIndex++) {
			var cell = table.rows[mergingRowIndex].cells[colIndex];
			if (cell && canMergeCells(table.rows, mergingRowIndex - 1, colIndex, mergingRowIndex, colIndex)) {
				mutateBlock(cell).spanAbove = true;
				var newSelectedRow = mergingRowIndex;
				while ((_b$1 = (_a$5 = table.rows[newSelectedRow]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[colIndex]) === null || _b$1 === void 0 ? void 0 : _b$1.spanAbove) {
					mutateBlock(table.rows[newSelectedRow].cells[colIndex]);
					newSelectedRow--;
				}
				var newCell = (_c$1 = table.rows[newSelectedRow]) === null || _c$1 === void 0 ? void 0 : _c$1.cells[colIndex];
				if (newCell) mutateBlock(newCell).isSelected = true;
			}
		}
	}
}
var MIN_WIDTH = 30;
function splitTableCellHorizontally(table) {
	var sel = getSelectedCells(table);
	if (sel) {
		var _loop_1 = function(colIndex$1) {
			if (table.rows.every(function(row, rowIndex) {
				var _a$5;
				return rowIndex < sel.firstRow || rowIndex > sel.lastRow || ((_a$5 = row.cells[colIndex$1 + 1]) === null || _a$5 === void 0 ? void 0 : _a$5.spanLeft);
			})) table.rows.forEach(function(row, rowIndex) {
				mutateBlock(row.cells[colIndex$1]);
				if (rowIndex >= sel.firstRow && rowIndex <= sel.lastRow) mutateBlock(row.cells[colIndex$1 + 1]).spanLeft = false;
			});
			else {
				table.rows.forEach(function(row, rowIndex) {
					var cell = row.cells[colIndex$1];
					if (cell) {
						var mutableCell = mutateBlock(cell);
						delete mutableCell.format.width;
						var newCell = createTableCell(cell.spanLeft, cell.spanAbove, cell.isHeader, mutableCell.format);
						newCell.dataset = __assign({}, cell.dataset);
						if (rowIndex < sel.firstRow || rowIndex > sel.lastRow) newCell.spanLeft = true;
						else newCell.isSelected = cell.isSelected;
						row.cells.splice(colIndex$1 + 1, 0, newCell);
						mutateBlock(row.cells[colIndex$1]);
					}
				});
				var newWidth = Math.max(table.widths[colIndex$1] / 2, MIN_WIDTH);
				table.widths.splice(colIndex$1, 1, newWidth, newWidth);
			}
		};
		for (var colIndex = sel.lastColumn; colIndex >= sel.firstColumn; colIndex--) _loop_1(colIndex);
	}
}
var MIN_HEIGHT = 22;
function splitTableCellVertically(table) {
	var sel = getSelectedCells(table);
	if (sel) for (var rowIndex = sel.lastRow; rowIndex >= sel.firstRow; rowIndex--) {
		var row = table.rows[rowIndex];
		var belowRow = table.rows[rowIndex + 1];
		row.cells.forEach(mutateBlock);
		if (belowRow === null || belowRow === void 0 ? void 0 : belowRow.cells.every(function(belowCell, colIndex) {
			return colIndex < sel.firstColumn || colIndex > sel.lastColumn || belowCell.spanAbove;
		})) belowRow.cells.forEach(function(belowCell, colIndex) {
			if (colIndex >= sel.firstColumn && colIndex <= sel.lastColumn) mutateBlock(belowCell).spanAbove = false;
		});
		else {
			var newHeight = Math.max(row.height /= 2, MIN_HEIGHT);
			var newRow = {
				format: __assign({}, row.format),
				height: newHeight,
				cells: row.cells.map(function(cell, colIndex) {
					var mutableCell = mutateBlock(cell);
					delete mutableCell.format.height;
					var newCell = createTableCell(cell.spanLeft, cell.spanAbove, cell.isHeader, mutableCell.format);
					newCell.dataset = __assign({}, cell.dataset);
					if (colIndex < sel.firstColumn || colIndex > sel.lastColumn) newCell.spanAbove = true;
					else newCell.isSelected = cell.isSelected;
					return newCell;
				})
			};
			row.height = newHeight;
			table.rows.splice(rowIndex + 1, 0, newRow);
		}
	}
}
var TextAlignValueMap = {
	alignCellLeft: "start",
	alignCellCenter: "center",
	alignCellRight: "end"
};
var VerticalAlignValueMap = {
	alignCellTop: "top",
	alignCellMiddle: "middle",
	alignCellBottom: "bottom"
};
function alignTableCellHorizontally(table, operation) {
	alignTableCellInternal(table, function(cell) {
		cell.format.textAlign = TextAlignValueMap[operation];
	});
}
function alignTableCellVertically(table, operation) {
	alignTableCellInternal(table, function(cell) {
		cell.format.verticalAlign = VerticalAlignValueMap[operation];
		updateTableCellMetadata(cell, function(metadata) {
			metadata = metadata || {};
			metadata.vAlignOverride = true;
			return metadata;
		});
	});
}
function alignTableCellInternal(table, callback) {
	var _a$5;
	var sel = getSelectedCells(table);
	if (sel) for (var rowIndex = sel.firstRow; rowIndex <= sel.lastRow; rowIndex++) for (var colIndex = sel.firstColumn; colIndex <= sel.lastColumn; colIndex++) {
		var cell = (_a$5 = table.rows[rowIndex]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[colIndex];
		if (cell === null || cell === void 0 ? void 0 : cell.format) {
			callback(mutateBlock(cell));
			cell.blocks.forEach(function(block) {
				if (block.blockType === "Paragraph" && block.format.textAlign) delete mutateBlock(block).format.textAlign;
			});
		}
	}
}
function editTable(editor, operation) {
	editor.focus();
	fixUpSafariSelection(editor);
	formatTableWithContentModel(editor, "editTable", function(tableModel) {
		switch (operation) {
			case "alignCellLeft":
			case "alignCellCenter":
			case "alignCellRight":
				alignTableCellHorizontally(tableModel, operation);
				break;
			case "alignCellTop":
			case "alignCellMiddle":
			case "alignCellBottom":
				alignTableCellVertically(tableModel, operation);
				break;
			case "alignCenter":
			case "alignLeft":
			case "alignRight":
				alignTable(tableModel, operation);
				break;
			case "deleteColumn":
				deleteTableColumn(tableModel);
				break;
			case "deleteRow":
				deleteTableRow(tableModel);
				break;
			case "deleteTable":
				deleteTable(tableModel);
				break;
			case "insertAbove":
			case "insertBelow":
				insertTableRow(tableModel, operation);
				break;
			case "insertLeft":
			case "insertRight":
				insertTableColumn(tableModel, operation);
				break;
			case "mergeAbove":
			case "mergeBelow":
				mergeTableRow(tableModel, operation);
				break;
			case "mergeCells":
				mergeTableCells(tableModel);
				break;
			case "mergeLeft":
			case "mergeRight":
				mergeTableColumn(tableModel, operation);
				break;
			case "splitHorizontally":
				splitTableCellHorizontally(tableModel);
				break;
			case "splitVertically":
				splitTableCellVertically(tableModel);
				break;
		}
	});
}
function fixUpSafariSelection(editor) {
	if (editor.getEnvironment().isSafari) {
		var selection = editor.getDOMSelection();
		if ((selection === null || selection === void 0 ? void 0 : selection.type) == "range" && !selection.range.collapsed) {
			selection.range.collapse(true);
			editor.setDOMSelection({
				type: "range",
				range: selection.range,
				isReverted: false
			});
		}
	}
}
function splitSelectedParagraphByBr(model) {
	var e_1, _a$5, e_2, _b$1, _c$1;
	var selections = getSelectedSegmentsAndParagraphs(model, false, false);
	try {
		for (var selections_1 = __values(selections), selections_1_1 = selections_1.next(); !selections_1_1.done; selections_1_1 = selections_1.next()) {
			var _d$1 = __read(selections_1_1.value, 3);
			_d$1[0];
			var para = _d$1[1], path = _d$1[2];
			if (para === null || para === void 0 ? void 0 : para.segments.some(function(s$1) {
				return s$1.segmentType == "Br";
			})) {
				var currentParagraph = shallowColonParagraph(para);
				var hasVisibleSegment = false;
				var newParagraphs = [];
				var parent_1 = mutateBlock(path[0]);
				var index$1 = parent_1.blocks.indexOf(para);
				if (index$1 >= 0) {
					try {
						for (var _e$1 = (e_2 = void 0, __values(mutateBlock(para).segments)), _f$1 = _e$1.next(); !_f$1.done; _f$1 = _e$1.next()) {
							var segment = _f$1.value;
							if (segment.segmentType == "Br") {
								if (!hasVisibleSegment) currentParagraph.segments.push(segment);
								if (currentParagraph.segments.length > 0) newParagraphs.push(currentParagraph);
								currentParagraph = shallowColonParagraph(para);
								hasVisibleSegment = false;
							} else {
								currentParagraph.segments.push(segment);
								if (segment.segmentType != "SelectionMarker") hasVisibleSegment = true;
							}
						}
					} catch (e_2_1) {
						e_2 = { error: e_2_1 };
					} finally {
						try {
							if (_f$1 && !_f$1.done && (_b$1 = _e$1.return)) _b$1.call(_e$1);
						} finally {
							if (e_2) throw e_2.error;
						}
					}
					if (currentParagraph.segments.length > 0) newParagraphs.push(currentParagraph);
					(_c$1 = parent_1.blocks).splice.apply(_c$1, __spreadArray([index$1, 1], __read(newParagraphs), false));
				}
			}
		}
	} catch (e_1_1) {
		e_1 = { error: e_1_1 };
	} finally {
		try {
			if (selections_1_1 && !selections_1_1.done && (_a$5 = selections_1.return)) _a$5.call(selections_1);
		} finally {
			if (e_1) throw e_1.error;
		}
	}
}
function shallowColonParagraph(para) {
	return createParagraph(false, para.format, para.segmentFormat, para.decorator);
}
function setListType(model, listType, removeMargins) {
	if (removeMargins === void 0) removeMargins = false;
	splitSelectedParagraphByBr(model);
	var paragraphOrListItems = getOperationalBlocks(model, ["ListItem"], []);
	var alreadyInExpectedType = paragraphOrListItems.every(function(_a$5) {
		var _b$1;
		var block = _a$5.block;
		return isBlockGroupOfType(block, "ListItem") ? ((_b$1 = block.levels[block.levels.length - 1]) === null || _b$1 === void 0 ? void 0 : _b$1.listType) == listType : shouldIgnoreBlock(block);
	});
	var existingListItems = [];
	paragraphOrListItems.forEach(function(_a$5, itemIndex) {
		var _b$1, _c$1;
		var block = _a$5.block, parent = _a$5.parent;
		if (isBlockGroupOfType(block, "ListItem")) {
			var level = block.levels.pop();
			if (!alreadyInExpectedType && level) {
				level.listType = listType;
				updateListMetadata(level, function(metadata) {
					return Object.assign({}, metadata, { applyListStyleFromLevel: true });
				});
				block.levels.push(level);
			} else if (block.blocks.length == 1) setParagraphNotImplicit(block.blocks[0]);
			if (alreadyInExpectedType) block.blocks.forEach(function(x$1) {
				copyFormat(x$1.format, block.format, ListFormats);
			});
		} else {
			var index$1 = parent.blocks.indexOf(block);
			if (index$1 >= 0) if (paragraphOrListItems.length == 1 || !shouldIgnoreBlock(block)) {
				var prevBlock = parent.blocks[index$1 - 1];
				var segmentFormat = block.blockType == "Paragraph" && ((_b$1 = block.segments[0]) === null || _b$1 === void 0 ? void 0 : _b$1.format) || {};
				var newListItem = createListItem([createListLevel(listType, {
					startNumberOverride: itemIndex > 0 || (prevBlock === null || prevBlock === void 0 ? void 0 : prevBlock.blockType) == "BlockGroup" && prevBlock.blockGroupType == "ListItem" && ((_c$1 = prevBlock.levels[0]) === null || _c$1 === void 0 ? void 0 : _c$1.listType) == "OL" ? void 0 : 1,
					direction: block.format.direction,
					textAlign: block.format.textAlign,
					marginBottom: removeMargins ? "0px" : void 0,
					marginTop: removeMargins ? "0px" : void 0
				})], {
					fontFamily: segmentFormat.fontFamily,
					fontSize: segmentFormat.fontSize,
					textColor: segmentFormat.textColor
				});
				if (block.blockType == "Paragraph") setParagraphNotImplicit(block);
				var mutableBlock = mutateBlock(block);
				newListItem.blocks.push(mutableBlock);
				adjustListIndentation(newListItem);
				copyFormat(newListItem.format, mutableBlock.format, ListFormatsToMove, true);
				copyFormat(newListItem.format, mutableBlock.format, ListFormatsToKeep);
				mutateBlock(parent).blocks.splice(index$1, 1, newListItem);
				existingListItems.push(newListItem);
				var levelIndex = newListItem.levels.length - 1;
				var level = mutateBlock(newListItem).levels[levelIndex];
				if (level) updateListMetadata(level, function(metadata) {
					return Object.assign({}, metadata, { applyListStyleFromLevel: true });
				});
			} else {
				existingListItems.forEach(function(x$1) {
					return mutateBlock(x$1).levels[0].format.marginBottom = "0px";
				});
				existingListItems = [];
			}
		}
	});
	normalizeContentModel(model);
	return paragraphOrListItems.length > 0;
}
function shouldIgnoreBlock(block) {
	switch (block.blockType) {
		case "Table": return false;
		case "Paragraph": return block.segments.every(function(x$1) {
			return x$1.segmentType == "Br" || x$1.segmentType == "SelectionMarker";
		});
		default: return true;
	}
}
function toggleBullet(editor, removeMargins) {
	if (removeMargins === void 0) removeMargins = false;
	editor.focus();
	editor.formatContentModel(function(model, context) {
		context.newPendingFormat = "preserve";
		return setListType(model, "UL", removeMargins);
	}, { apiName: "toggleBullet" });
}
function toggleNumbering(editor, removeMargins) {
	if (removeMargins === void 0) removeMargins = false;
	editor.focus();
	editor.formatContentModel(function(model, context) {
		context.newPendingFormat = "preserve";
		return setListType(model, "OL", removeMargins);
	}, { apiName: "toggleNumbering" });
}
function adjustWordSelection(model, marker) {
	var markerBlock;
	iterateSelections(model, function(_, __, block, segments$1) {
		if ((block === null || block === void 0 ? void 0 : block.blockType) == "Paragraph" && (segments$1 === null || segments$1 === void 0 ? void 0 : segments$1.length) == 1 && segments$1[0] == marker) markerBlock = mutateBlock(block);
		return true;
	});
	var tempSegments = markerBlock ? __spreadArray([], __read(markerBlock.segments), false) : void 0;
	if (tempSegments && markerBlock) {
		var segments = [];
		var markerSelectionIndex = tempSegments.indexOf(marker);
		for (var i$1 = markerSelectionIndex - 1; i$1 >= 0; i$1--) {
			var currentSegment = tempSegments[i$1];
			if (currentSegment.segmentType == "Text") {
				var found = findDelimiter(currentSegment, false);
				if (found > -1) {
					if (found == currentSegment.text.length) break;
					splitTextSegment$1(tempSegments, currentSegment, i$1, found);
					segments.push(tempSegments[i$1 + 1]);
					break;
				} else segments.push(tempSegments[i$1]);
			} else break;
		}
		markerSelectionIndex = tempSegments.indexOf(marker);
		segments.push(marker);
		if (segments.length <= 1) return segments;
		for (var i$1 = markerSelectionIndex + 1; i$1 < tempSegments.length; i$1++) {
			var currentSegment = tempSegments[i$1];
			if (currentSegment.segmentType == "Text") {
				var found = findDelimiter(currentSegment, true);
				if (found > -1) {
					if (found == 0) break;
					splitTextSegment$1(tempSegments, currentSegment, i$1, found);
					segments.push(tempSegments[i$1]);
					break;
				} else segments.push(tempSegments[i$1]);
			} else break;
		}
		if (segments[segments.length - 1] == marker) return [marker];
		markerBlock.segments = tempSegments;
		return segments;
	} else return [marker];
}
function findDelimiter(segment, moveRightward) {
	var word = segment.text;
	var offset = -1;
	if (moveRightward) for (var i$1 = 0; i$1 < word.length; i$1++) {
		var char = word[i$1];
		if (isPunctuation(char) || isSpace(char)) {
			offset = i$1;
			break;
		}
	}
	else for (var i$1 = word.length - 1; i$1 >= 0; i$1--) {
		var char = word[i$1];
		if (isPunctuation(char) || isSpace(char)) {
			offset = i$1 + 1;
			break;
		}
	}
	return offset;
}
function splitTextSegment$1(segments, textSegment, index$1, found) {
	var text = textSegment.text;
	var newSegmentLeft = createText(text.substring(0, found), textSegment.format, textSegment.link, textSegment.code);
	var newSegmentRight = createText(text.substring(found, text.length), textSegment.format, textSegment.link, textSegment.code);
	segments.splice(index$1, 1, newSegmentLeft, newSegmentRight);
}
function createEditorContextForEntity(editor, entity) {
	var _a$5;
	var domHelper = editor.getDOMHelper();
	var context = {
		isDarkMode: editor.isDarkMode(),
		defaultFormat: __assign({}, entity.format),
		darkColorHandler: editor.getColorManager(),
		addDelimiterForEntity: false,
		allowCacheElement: false,
		domIndexer: void 0,
		zoomScale: domHelper.calculateZoomScale(),
		experimentalFeatures: []
	};
	if (((_a$5 = editor.getDocument().defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getComputedStyle(entity.wrapper).direction) == "rtl") context.isRootRtl = true;
	return context;
}
function formatSegmentWithContentModel(editor, apiName, toggleStyleCallback, segmentHasStyleCallback, includingFormatHolder, afterFormatCallback) {
	editor.formatContentModel(function(model, context) {
		var segmentAndParagraphs = getSelectedSegmentsAndParagraphs(model, !!includingFormatHolder, true, true);
		var isCollapsedSelection = segmentAndParagraphs.length >= 1 && segmentAndParagraphs.every(function(x$1) {
			return x$1[0].segmentType == "SelectionMarker";
		});
		if (isCollapsedSelection) {
			var para_1 = segmentAndParagraphs[0][1];
			var path_1 = segmentAndParagraphs[0][2];
			segmentAndParagraphs = adjustWordSelection(model, segmentAndParagraphs[0][0]).map(function(x$1) {
				return [
					x$1,
					para_1,
					path_1
				];
			});
			if (segmentAndParagraphs.length > 1) isCollapsedSelection = false;
		}
		var formatsAndSegments = [];
		var modelsFromEntities = [];
		segmentAndParagraphs.forEach(function(item) {
			if (item[0].segmentType == "Entity") expandEntitySelections(editor, item[0], formatsAndSegments, modelsFromEntities);
			else formatsAndSegments.push([
				item[0].format,
				item[0],
				item[1]
			]);
		});
		var isTurningOff = segmentHasStyleCallback ? formatsAndSegments.every(function(_a$5) {
			var _b$1 = __read(_a$5, 3), format = _b$1[0], segment = _b$1[1], paragraph = _b$1[2];
			return segmentHasStyleCallback(format, segment, paragraph);
		}) : false;
		formatsAndSegments.forEach(function(_a$5) {
			var _b$1 = __read(_a$5, 3), format = _b$1[0], segment = _b$1[1], paragraph = _b$1[2];
			toggleStyleCallback(format, !isTurningOff, segment, paragraph);
		});
		afterFormatCallback === null || afterFormatCallback === void 0 || afterFormatCallback(model, isTurningOff, context);
		formatsAndSegments.forEach(function(_a$5) {
			var _b$1 = __read(_a$5, 3);
			_b$1[0];
			_b$1[1];
			var paragraph = _b$1[2];
			if (paragraph) mergeTextSegments(paragraph);
		});
		writeBackEntities(editor, modelsFromEntities);
		if (isCollapsedSelection) {
			context.newPendingFormat = segmentAndParagraphs[0][0].format;
			editor.focus();
			return false;
		} else return formatsAndSegments.length > 0;
	}, { apiName });
}
function expandEntitySelections(editor, entity, formatsAndSegments, modelsFromEntities) {
	var _a$5 = entity.entityFormat, id = _a$5.id, type = _a$5.entityType, isReadonly = _a$5.isReadonly;
	if (id && type) {
		var formattableRoots = [];
		var entityOperationEventData = {
			entity: {
				id,
				type,
				isReadonly: !!isReadonly,
				wrapper: entity.wrapper
			},
			operation: "beforeFormat",
			formattableRoots
		};
		editor.triggerEvent("entityOperation", entityOperationEventData);
		formattableRoots.forEach(function(root$12) {
			if (entity.wrapper.contains(root$12.element)) {
				var context = createDomToModelContext(createEditorContextForEntity(editor, entity), root$12.domToModelOptions);
				context.isInSelection = true;
				var model = domToContentModel(root$12.element, context);
				getSelectedSegmentsAndParagraphs(model, false, false, true).forEach(function(item) {
					formatsAndSegments.push([
						item[0].format,
						item[0],
						item[1]
					]);
				});
				modelsFromEntities.push([
					entity,
					root$12,
					model
				]);
			}
		});
	}
}
function writeBackEntities(editor, modelsFromEntities) {
	modelsFromEntities.forEach(function(_a$5) {
		var _b$1 = __read(_a$5, 3), entity = _b$1[0], root$12 = _b$1[1], model = _b$1[2];
		var modelToDomContext = createModelToDomContext(createEditorContextForEntity(editor, entity), root$12.modelToDomOptions);
		contentModelToDom(editor.getDocument(), root$12.element, model, modelToDomContext);
	});
}
function toggleBold(editor, options) {
	editor.focus();
	formatSegmentWithContentModel(editor, "toggleBold", function(format, isTurningOn) {
		format.fontWeight = isTurningOn ? "bold" : "normal";
	}, function(format, _, paragraph) {
		var _a$5;
		return isBold(typeof format.fontWeight == "undefined" ? (_a$5 = paragraph === null || paragraph === void 0 ? void 0 : paragraph.decorator) === null || _a$5 === void 0 ? void 0 : _a$5.format.fontWeight : format.fontWeight);
	}, void 0, function(_model, isTurningOff, context) {
		if (options === null || options === void 0 ? void 0 : options.announceFormatChange) context.announceData = { defaultStrings: isTurningOff ? "announceBoldOff" : "announceBoldOn" };
	});
}
function toggleItalic(editor, options) {
	editor.focus();
	formatSegmentWithContentModel(editor, "toggleItalic", function(format, isTurningOn) {
		format.italic = !!isTurningOn;
	}, function(format) {
		return !!format.italic;
	}, void 0, function(_model, isTurningOff, context) {
		if (options === null || options === void 0 ? void 0 : options.announceFormatChange) context.announceData = { defaultStrings: isTurningOff ? "announceItalicOff" : "announceItalicOn" };
	});
}
function toggleUnderline(editor, options) {
	editor.focus();
	formatSegmentWithContentModel(editor, "toggleUnderline", function(format, isTurningOn, segment) {
		format.underline = !!isTurningOn;
		if (segment === null || segment === void 0 ? void 0 : segment.link) segment.link.format.underline = !!isTurningOn;
	}, function(format, segment) {
		var _a$5, _b$1;
		return !!format.underline || !!((_b$1 = (_a$5 = segment === null || segment === void 0 ? void 0 : segment.link) === null || _a$5 === void 0 ? void 0 : _a$5.format) === null || _b$1 === void 0 ? void 0 : _b$1.underline);
	}, false, function(_model, isTurningOff, context) {
		if (options === null || options === void 0 ? void 0 : options.announceFormatChange) context.announceData = { defaultStrings: isTurningOff ? "announceUnderlineOff" : "announceUnderlineOn" };
	});
}
function toggleStrikethrough(editor) {
	editor.focus();
	formatSegmentWithContentModel(editor, "toggleStrikethrough", function(format, isTurningOn) {
		format.strikethrough = !!isTurningOn;
	}, function(format) {
		return !!format.strikethrough;
	});
}
function setBackgroundColor(editor, backgroundColor) {
	editor.focus();
	var lastParagraph = null;
	var lastSegmentIndex = -1;
	formatSegmentWithContentModel(editor, "setBackgroundColor", function(format, _, segment, paragraph) {
		if (backgroundColor === null) delete format.backgroundColor;
		else format.backgroundColor = backgroundColor;
		if (segment && paragraph && segment.segmentType != "SelectionMarker") {
			lastParagraph = paragraph;
			lastSegmentIndex = lastParagraph.segments.indexOf(segment);
		}
	}, void 0, void 0, function(model) {
		var _a$5;
		if (lastParagraph && lastSegmentIndex >= 0) {
			var marker = createSelectionMarker((_a$5 = lastParagraph.segments[lastSegmentIndex]) === null || _a$5 === void 0 ? void 0 : _a$5.format);
			lastParagraph.segments.splice(lastSegmentIndex + 1, 0, marker);
			setSelection(model, marker, marker);
		}
	});
}
function setFontSize(editor, fontSize) {
	editor.focus();
	formatSegmentWithContentModel(editor, "setFontSize", function(format, _, __, paragraph) {
		return setFontSizeInternal(fontSize, format, paragraph);
	}, void 0, true);
}
function setFontSizeInternal(fontSize, format, paragraph) {
	var _a$5;
	format.fontSize = fontSize;
	if ((_a$5 = paragraph === null || paragraph === void 0 ? void 0 : paragraph.segmentFormat) === null || _a$5 === void 0 ? void 0 : _a$5.fontSize) {
		var size_1 = paragraph.segmentFormat.fontSize;
		paragraph.segments.forEach(function(segment) {
			if (!segment.format.fontSize) segment.format.fontSize = size_1;
		});
		delete paragraph.segmentFormat.fontSize;
	}
}
function setTextColor(editor, textColor) {
	editor.focus();
	formatSegmentWithContentModel(editor, "setTextColor", textColor === null ? function(format, _, segment) {
		delete format.textColor;
		if (segment === null || segment === void 0 ? void 0 : segment.link) delete segment.link.format.textColor;
	} : function(format, _, segment) {
		format.textColor = textColor;
		if (segment === null || segment === void 0 ? void 0 : segment.link) segment.link.format.textColor = textColor;
	}, void 0, true);
}
var FONT_SIZES = [
	8,
	9,
	10,
	11,
	12,
	14,
	16,
	18,
	20,
	22,
	24,
	26,
	28,
	36,
	48,
	72
];
var MIN_FONT_SIZE = 1;
var MAX_FONT_SIZE = 1e3;
function changeFontSize(editor, change, fontSizes) {
	if (fontSizes === void 0) fontSizes = FONT_SIZES;
	editor.focus();
	formatSegmentWithContentModel(editor, "changeFontSize", function(format, _, __, paragraph) {
		return changeFontSizeInternal(change, format, paragraph, fontSizes);
	}, void 0, true);
}
function changeFontSizeInternal(change, format, paragraph, fontSizes) {
	if (format.fontSize) {
		var sizeInPt = parseValueWithUnit(format.fontSize, void 0, "pt");
		if (sizeInPt > 0) setFontSizeInternal(getNewFontSize(sizeInPt, change == "increase" ? 1 : -1, fontSizes) + "pt", format, paragraph);
	}
}
function getNewFontSize(pt, changeBase, fontSizes) {
	pt = changeBase == 1 ? Math.floor(pt) : Math.ceil(pt);
	var last = fontSizes[fontSizes.length - 1];
	if (pt <= fontSizes[0]) pt = Math.max(pt + changeBase, MIN_FONT_SIZE);
	else if (pt > last || pt == last && changeBase == 1) {
		pt = pt / 10;
		pt = changeBase == 1 ? Math.floor(pt) : Math.ceil(pt);
		pt = Math.min(Math.max((pt + changeBase) * 10, last), MAX_FONT_SIZE);
	} else if (changeBase == 1) {
		for (var i$1 = 0; i$1 < fontSizes.length; i$1++) if (pt < fontSizes[i$1]) {
			pt = fontSizes[i$1];
			break;
		}
	} else for (var i$1 = fontSizes.length - 1; i$1 >= 0; i$1--) if (pt > fontSizes[i$1]) {
		pt = fontSizes[i$1];
		break;
	}
	return pt;
}
function splitTextSegment(textSegment, parent, start, end) {
	var _a$5;
	var text = textSegment.text;
	var index$1 = parent.segments.indexOf(textSegment);
	var middleSegment = createText(text.substring(start, end), textSegment.format, textSegment.link, textSegment.code);
	var newSegments = [middleSegment];
	if (start > 0) newSegments.unshift(createText(text.substring(0, start), textSegment.format, textSegment.link, textSegment.code));
	if (end < text.length) newSegments.push(createText(text.substring(end), textSegment.format, textSegment.link, textSegment.code));
	newSegments.forEach(function(segment) {
		return segment.isSelected = textSegment.isSelected;
	});
	(_a$5 = parent.segments).splice.apply(_a$5, __spreadArray([index$1, 1], __read(newSegments), false));
	return middleSegment;
}
function findListItemsInSameThread(group, currentItem) {
	var items = [];
	findListItems(group, items);
	return filterListItems(items, currentItem);
}
function findListItems(group, result) {
	group.blocks.forEach(function(block) {
		switch (block.blockType) {
			case "BlockGroup":
				if (block.blockGroupType == "ListItem") result.push(block);
				else {
					pushNullIfNecessary(result);
					findListItems(block, result);
					pushNullIfNecessary(result);
				}
				break;
			case "Paragraph":
				pushNullIfNecessary(result);
				block.segments.forEach(function(segment) {
					if (segment.segmentType == "General") findListItems(segment, result);
				});
				pushNullIfNecessary(result);
				break;
			case "Table":
				pushNullIfNecessary(result);
				block.rows.forEach(function(row) {
					return row.cells.forEach(function(cell) {
						findListItems(cell, result);
					});
				});
				pushNullIfNecessary(result);
				break;
		}
	});
}
function pushNullIfNecessary(result) {
	var last = result[result.length - 1];
	if (!last || last !== null) result.push(null);
}
function filterListItems(items, currentItem) {
	var _a$5;
	var result = [];
	var currentIndex = items.indexOf(currentItem);
	var levelLength = currentItem.levels.length;
	var isOrderedList = ((_a$5 = currentItem.levels[levelLength - 1]) === null || _a$5 === void 0 ? void 0 : _a$5.listType) == "OL";
	if (currentIndex >= 0) {
		for (var i$1 = currentIndex; i$1 >= 0; i$1--) {
			var item = items[i$1];
			if (!item) if (isOrderedList) continue;
			else break;
			var startNumberOverride = hasStartNumberOverride(item, levelLength);
			if (areListTypesCompatible(items, currentIndex, i$1)) {
				result.unshift(item);
				if (isOrderedList && startNumberOverride) break;
			} else if (!isOrderedList || startNumberOverride || item.levels.length < currentItem.levels.length) break;
		}
		for (var i$1 = currentIndex + 1; i$1 < items.length; i$1++) {
			var item = items[i$1];
			if (!item) if (isOrderedList) continue;
			else break;
			var startNumberOverride = hasStartNumberOverride(item, levelLength);
			if (areListTypesCompatible(items, currentIndex, i$1) && !startNumberOverride) result.push(item);
			else if (!isOrderedList || startNumberOverride || item.levels.length < currentItem.levels.length) break;
		}
	}
	return result;
}
function areListTypesCompatible(listItems, currentIndex, compareToIndex) {
	var currentLevels = listItems[currentIndex].levels;
	var compareToLevels = listItems[compareToIndex].levels;
	return currentLevels.length <= compareToLevels.length && currentLevels.every(function(currentLevel, i$1) {
		return currentLevel.listType == compareToLevels[i$1].listType;
	});
}
function hasStartNumberOverride(item, levelLength) {
	return item.levels.slice(0, levelLength).some(function(level) {
		return level.format.startNumberOverride !== void 0;
	});
}
function setModelListStyle(model, style) {
	var listItem = getFirstSelectedListItem(model);
	if (listItem) {
		var listItems = findListItemsInSameThread(model, listItem);
		var levelIndex_1 = listItem.levels.length - 1;
		listItems.forEach(function(listItem$1) {
			var level = mutateBlock(listItem$1).levels[levelIndex_1];
			if (level) updateListMetadata(level, function(metadata) {
				return Object.assign({}, metadata, style);
			});
		});
	}
	return !!listItem;
}
function setModelListStartNumber(model, value) {
	var listItem = getFirstSelectedListItem(model);
	var level = listItem ? mutateBlock(listItem).levels[(listItem === null || listItem === void 0 ? void 0 : listItem.levels.length) - 1] : null;
	if (level) {
		level.format.startNumberOverride = value;
		return true;
	} else return false;
}
function getListAnnounceData(path) {
	var index$1 = getClosestAncestorBlockGroupIndex(path, ["ListItem"], ["TableCell"]);
	if (index$1 >= 0) {
		var listItem = path[index$1];
		var level = listItem.levels[listItem.levels.length - 1];
		if (!level || level.format.displayForDummyItem) return null;
		else if (level.listType == "OL") {
			var listNumber = getListNumber(path, listItem);
			var metadata = getListMetadata(level);
			var listStyle = getAutoListStyleType("OL", metadata !== null && metadata !== void 0 ? metadata : {}, listItem.levels.length - 1, level.format.listStyleType);
			return listStyle === void 0 ? null : {
				defaultStrings: "announceListItemNumbering",
				formatStrings: [getOrderedListNumberStr(listStyle, listNumber)]
			};
		} else return { defaultStrings: "announceListItemBullet" };
	} else return null;
}
function getListNumber(path, listItem) {
	var _a$5, _b$1;
	var items = findListItemsInSameThread(path[path.length - 1], listItem);
	var listNumber = 0;
	for (var i$1 = 0; i$1 < items.length; i$1++) {
		var item = items[i$1];
		if (listNumber == 0 && item.levels.length == listItem.levels.length) listNumber = (_b$1 = (_a$5 = item.levels[item.levels.length - 1]) === null || _a$5 === void 0 ? void 0 : _a$5.format.startNumberOverride) !== null && _b$1 !== void 0 ? _b$1 : 1;
		if (item == listItem) break;
		else if (item.levels.length < listItem.levels.length) listNumber = 0;
		else if (item.levels.length > listItem.levels.length) continue;
		else if (!item.levels[item.levels.length - 1].format.displayForDummyItem) listNumber++;
	}
	return listNumber;
}
function setModelIndentation(model, indentation, length, context) {
	if (length === void 0) length = 40;
	splitSelectedParagraphByBr(model);
	var paragraphOrListItem = getOperationalBlocks(model, ["ListItem"], ["TableCell"]);
	var isIndent = indentation == "indent";
	var modifiedBlocks = [];
	paragraphOrListItem.forEach(function(_a$5) {
		var block = _a$5.block, parent = _a$5.parent, path = _a$5.path;
		if (isBlockGroupOfType(block, "ListItem")) {
			var thread = findListItemsInSameThread(model, block);
			var firstItem = thread[0];
			if (isSelected(firstItem) && firstItem.levels.length == 1) {
				var level = block.levels[0];
				var format = level.format;
				var marginLeft = format.marginLeft, marginRight = format.marginRight;
				var newValue = calculateMarginValue(format, isIndent, length);
				var isRtl = format.direction == "rtl";
				var originalValue = parseValueWithUnit(isRtl ? marginRight : marginLeft);
				if (!isIndent && originalValue == 0) block.levels.pop();
				else if (newValue !== null) if (isRtl) level.format.marginRight = newValue + "px";
				else level.format.marginLeft = newValue + "px";
			} else if (block.levels.length == 1 || !isMultilevelSelection(model, block, parent)) {
				if (isIndent) {
					var threadIdx = thread.indexOf(block);
					var previousItem = thread[threadIdx - 1];
					var nextItem = thread[threadIdx + 1];
					var levelLength = block.levels.length;
					var lastLevel = block.levels[levelLength - 1];
					var newLevel = createListLevel((lastLevel === null || lastLevel === void 0 ? void 0 : lastLevel.listType) || "UL", lastLevel === null || lastLevel === void 0 ? void 0 : lastLevel.format, previousItem && previousItem.levels.length > levelLength ? previousItem.levels[levelLength].dataset : nextItem && nextItem.levels.length > levelLength ? nextItem.levels[levelLength].dataset : void 0);
					updateListMetadata(newLevel, function(metadata) {
						metadata = metadata || {};
						metadata.applyListStyleFromLevel = true;
						return metadata;
					});
					delete newLevel.format.startNumberOverride;
					block.levels.push(newLevel);
				} else block.levels.pop();
				if (block.levels.length > 0 && context) context.announceData = getListAnnounceData(__spreadArray([block], __read(path), false));
			}
		} else if (block) {
			var currentBlock = block;
			var currentParent = parent;
			while (currentParent && modifiedBlocks.indexOf(currentBlock) < 0) {
				var index$1 = path.indexOf(currentParent);
				var format = mutateBlock(currentBlock).format;
				var newValue = calculateMarginValue(format, isIndent, length);
				if (newValue !== null) {
					var isRtl = format.direction == "rtl";
					if (isRtl) format.marginRight = newValue + "px";
					else format.marginLeft = newValue + "px";
					modifiedBlocks.push(currentBlock);
					break;
				} else if (currentParent.blockGroupType == "FormatContainer" && index$1 >= 0) {
					mutateBlock(currentParent);
					currentBlock = currentParent;
					currentParent = path[index$1 + 1];
				} else break;
			}
		}
	});
	return paragraphOrListItem.length > 0;
}
function isSelected(listItem) {
	return listItem.blocks.some(function(block) {
		if (block.blockType == "Paragraph") return block.segments.some(function(segment) {
			return segment.isSelected;
		});
	});
}
function isMultilevelSelection(model, listItem, parent) {
	for (var i$1 = parent.blocks.indexOf(listItem) - 1; i$1 >= 0; i$1--) {
		var block = parent.blocks[i$1];
		if (isBlockGroupOfType(block, "ListItem") && block.levels.length == 1 && isSelected(block)) {
			var firstItem = findListItemsInSameThread(model, block)[0];
			return isSelected(firstItem);
		}
		if (!isBlockGroupOfType(block, "ListItem")) return false;
	}
	return false;
}
function calculateMarginValue(format, isIndent, length) {
	if (length === void 0) length = 40;
	var marginLeft = format.marginLeft, marginRight = format.marginRight;
	var originalValue = parseValueWithUnit(format.direction == "rtl" ? marginRight : marginLeft);
	var newValue = (isIndent ? Math.ceil : Math.floor)(originalValue / length) * length;
	if (newValue == originalValue) newValue = Math.max(newValue + length * (isIndent ? 1 : -1), 0);
	if (newValue == originalValue) return null;
	else return newValue;
}
var ResultMap = {
	left: {
		ltr: "start",
		rtl: "end"
	},
	center: {
		ltr: "center",
		rtl: "center"
	},
	right: {
		ltr: "end",
		rtl: "start"
	},
	justify: {
		ltr: "justify",
		rtl: "justify"
	}
};
var TableAlignMap = {
	left: {
		ltr: "alignLeft",
		rtl: "alignRight"
	},
	center: {
		ltr: "alignCenter",
		rtl: "alignCenter"
	},
	right: {
		ltr: "alignRight",
		rtl: "alignLeft"
	}
};
function setModelAlignment(model, alignment) {
	splitSelectedParagraphByBr(model);
	var paragraphOrListItemOrTable = getOperationalBlocks(model, ["ListItem"], ["TableCell"]);
	paragraphOrListItemOrTable.forEach(function(_a$5) {
		var readonlyBlock = _a$5.block;
		var block = mutateBlock(readonlyBlock);
		var newAlignment = ResultMap[alignment][block.format.direction == "rtl" ? "rtl" : "ltr"];
		if (block.blockType === "Table" && alignment !== "justify") alignTable(block, TableAlignMap[alignment][block.format.direction == "rtl" ? "rtl" : "ltr"]);
		else if (block) {
			if (block.blockType === "BlockGroup" && block.blockGroupType === "ListItem") block.blocks.forEach(function(b$1) {
				var format$1 = mutateBlock(b$1).format;
				format$1.textAlign = newAlignment;
			});
			var format = block.format;
			format.textAlign = newAlignment;
		}
	});
	return paragraphOrListItemOrTable.length > 0;
}
function setAlignment(editor, alignment) {
	editor.focus();
	editor.formatContentModel(function(model) {
		return setModelAlignment(model, alignment);
	}, { apiName: "setAlignment" });
}
function reducedModelChildProcessor(group, parent, context) {
	if (!context.nodeStack) {
		var selectionRootNode = getSelectionRootNode(context.selection);
		context.nodeStack = selectionRootNode ? createNodeStack(parent, selectionRootNode) : [];
	}
	var stackChild = context.nodeStack.pop();
	if (stackChild) {
		var _a$5 = __read(getRegularSelectionOffsets(context, parent), 2), nodeStartOffset = _a$5[0], nodeEndOffset = _a$5[1];
		var index$1 = nodeStartOffset >= 0 || nodeEndOffset >= 0 ? getChildIndex(parent, stackChild) : -1;
		if (index$1 >= 0) handleRegularSelection(index$1, context, group, nodeStartOffset, nodeEndOffset);
		processChildNode(group, stackChild, context);
		if (index$1 >= 0) handleRegularSelection(index$1 + 1, context, group, nodeStartOffset, nodeEndOffset);
	} else context.defaultElementProcessors.child(group, parent, context);
}
function createNodeStack(root$12, startNode) {
	var result = [];
	var node = startNode;
	while (node && root$12 != node && root$12.contains(node)) {
		if (isNodeOfType(node, "ELEMENT_NODE") && node.tagName == "TABLE") result.splice(0, result.length, node);
		else result.push(node);
		node = node.parentNode;
	}
	return result;
}
function getChildIndex(parent, stackChild) {
	var index$1 = 0;
	var child$1 = parent.firstChild;
	while (child$1 && child$1 != stackChild) {
		index$1++;
		child$1 = child$1.nextSibling;
	}
	return index$1;
}
function getFormatState(editor, conflictSolution) {
	if (conflictSolution === void 0) conflictSolution = "remove";
	var pendingFormat = editor.getPendingFormat();
	var manager = editor.getSnapshotsManager();
	var result = {
		canUndo: manager.hasNewContent || manager.canMove(-1),
		canRedo: manager.canMove(1),
		isDarkMode: editor.isDarkMode()
	};
	editor.formatContentModel(function(model) {
		retrieveModelFormatState(model, pendingFormat, result, conflictSolution, editor.getDOMHelper(), editor.isDarkMode(), editor.getColorManager());
		return false;
	}, void 0, {
		processorOverride: { child: reducedModelChildProcessor },
		tryGetFromCache: true
	});
	return result;
}
function clearModelFormat(model, blocksToClear, segmentsToClear, tablesToClear) {
	var pendingStructureChange = false;
	iterateSelections(model, function(path$1, tableContext, block$1, segments) {
		if (segments) {
			if ((block$1 === null || block$1 === void 0 ? void 0 : block$1.blockType) == "Paragraph") {
				var mutableSegments = __read(mutateSegments(block$1, segments), 2)[1];
				segmentsToClear.push.apply(segmentsToClear, __spreadArray([], __read(mutableSegments), false));
			} else if (path$1[0].blockGroupType == "ListItem" && segments.length == 1 && path$1[0].formatHolder == segments[0]) segmentsToClear.push(mutateBlock(path$1[0]).formatHolder);
		}
		if (block$1) blocksToClear.push([path$1, mutateBlock(block$1)]);
		else if (tableContext) clearTableCellFormat(tableContext, tablesToClear);
	}, { includeListFormatHolder: model.format ? "never" : "anySegment" });
	var marker = segmentsToClear[0];
	if (blocksToClear.length == 1 && isOnlySelectionMarkerSelected(blocksToClear[0][1]) && blocksToClear.length == 1) {
		segmentsToClear.splice.apply(segmentsToClear, __spreadArray([0, segmentsToClear.length], __read(adjustWordSelection(model, marker)), false));
		pendingStructureChange = clearListFormat(blocksToClear[0][0]) || pendingStructureChange;
	} else if (blocksToClear.length > 1 || blocksToClear.some(function(x$1) {
		return isWholeBlockSelected(x$1[1]);
	})) for (var i$1 = blocksToClear.length - 1; i$1 >= 0; i$1--) {
		var _a$5 = __read(blocksToClear[i$1], 2), path = _a$5[0], block = _a$5[1];
		clearBlockFormat(path, block);
		pendingStructureChange = clearListFormat(path) || pendingStructureChange;
		clearContainerFormat(path, block);
	}
	clearSegmentsFormat(segmentsToClear, model.format);
	createTablesFormat(tablesToClear);
	return pendingStructureChange;
}
function createTablesFormat(tablesToClear) {
	tablesToClear.forEach(function(x$1) {
		var _a$5 = __read(x$1, 2), table = _a$5[0];
		if (_a$5[1]) {
			table.format = {
				useBorderBox: table.format.useBorderBox,
				borderCollapse: table.format.borderCollapse
			};
			updateTableMetadata(table, function() {
				return null;
			});
		}
		applyTableFormat(table, void 0, true);
	});
}
function clearSegmentsFormat(segmentsToClear, defaultSegmentFormat) {
	segmentsToClear.forEach(function(x$1) {
		x$1.format = __assign({}, defaultSegmentFormat || {});
		if (x$1.link) delete x$1.link.format.textColor;
		delete x$1.code;
	});
}
function clearTableCellFormat(tableContext, tablesToClear) {
	if (tableContext) {
		var table_1 = tableContext.table, colIndex = tableContext.colIndex, rowIndex = tableContext.rowIndex, isWholeTableSelected$1 = tableContext.isWholeTableSelected;
		var cell = table_1.rows[rowIndex].cells[colIndex];
		if (cell.isSelected) {
			var mutableCell = mutateBlock(cell);
			updateTableCellMetadata(mutableCell, function() {
				return null;
			});
			mutableCell.isHeader = false;
			mutableCell.format = { useBorderBox: cell.format.useBorderBox };
		}
		if (!tablesToClear.find(function(x$1) {
			return x$1[0] == table_1;
		})) tablesToClear.push([mutateBlock(table_1), isWholeTableSelected$1]);
	}
}
function clearContainerFormat(path, block) {
	var containerPathIndex = getClosestAncestorBlockGroupIndex(path, ["FormatContainer"], ["TableCell"]);
	if (containerPathIndex >= 0 && containerPathIndex < path.length - 1) {
		var container = mutateBlock(path[containerPathIndex]);
		var containerIndex = path[containerPathIndex + 1].blocks.indexOf(container);
		var blockIndex = container.blocks.indexOf(block);
		if (blockIndex >= 0 && containerIndex >= 0) {
			var newContainer = createFormatContainer(container.tagName, container.format);
			container.blocks.splice(blockIndex, 1);
			newContainer.blocks = container.blocks.splice(blockIndex);
			mutateBlock(path[containerPathIndex + 1]).blocks.splice(containerIndex + 1, 0, block, newContainer);
		}
	}
}
function clearListFormat(path) {
	var listItem = path[getClosestAncestorBlockGroupIndex(path, ["ListItem"], ["TableCell"])];
	if (listItem) {
		mutateBlock(listItem).levels = [];
		return true;
	} else return false;
}
function clearBlockFormat(path, block) {
	if (block.blockType == "Divider") {
		var index$1 = path[0].blocks.indexOf(block);
		if (index$1 >= 0) mutateBlock(path[0]).blocks.splice(index$1, 1);
	} else if (block.blockType == "Paragraph") {
		block.format = {};
		delete block.decorator;
	}
}
function isOnlySelectionMarkerSelected(block) {
	var segments = block.blockType == "Paragraph" ? block.segments.filter(function(x$1) {
		return x$1.isSelected;
	}) : [];
	return segments.length == 1 && segments[0].segmentType == "SelectionMarker";
}
function isWholeBlockSelected(block) {
	return block.isSelected || block.blockType == "Paragraph" && block.segments.every(function(x$1) {
		return x$1.isSelected;
	});
}
var MAX_TRY = 3;
function clearFormat(editor) {
	editor.focus();
	editor.formatContentModel(function(model) {
		var changed = false;
		var needtoRun = true;
		var triedTimes = 0;
		while (needtoRun && triedTimes++ < MAX_TRY) {
			var blocksToClear = [];
			var segmentsToClear = [];
			var tablesToClear = [];
			needtoRun = clearModelFormat(model, blocksToClear, segmentsToClear, tablesToClear);
			normalizeContentModel(model);
			changed = changed || blocksToClear.length > 0 || segmentsToClear.length > 0 || tablesToClear.length > 0;
		}
		return changed;
	}, { apiName: "clearFormat" });
}
var httpExcludeRegEx = /^[^?]+%[^0-9a-f]+|^[^?]+%[0-9a-f][^0-9a-f]+|^[^?]+%00|^[^?]+%$|^https?:\/\/[^?\/]+@|^www\.[^?\/]+@/i;
var labelRegEx = "[a-z0-9](?:[a-z0-9-]*[a-z0-9])?";
var domainPortWithUrlRegEx = "(?:" + labelRegEx + "\\.)*" + labelRegEx + "(?:\\:[0-9]+)?(?:[\\/\\?]\\S*)?";
var linkMatchRules = {
	http: {
		match: new RegExp("^(?:microsoft-edge:)?http:\\/\\/" + domainPortWithUrlRegEx + "|www\\." + domainPortWithUrlRegEx, "i"),
		except: httpExcludeRegEx,
		normalizeUrl: function(url) {
			return new RegExp("^(?:microsoft-edge:)?http:\\/\\/", "i").test(url) ? url : "http://" + url;
		}
	},
	https: {
		match: new RegExp("^(?:microsoft-edge:)?https:\\/\\/" + domainPortWithUrlRegEx, "i"),
		except: httpExcludeRegEx
	},
	mailto: { match: new RegExp("^mailto:\\S+@\\S+\\.\\S+", "i") },
	notes: { match: new RegExp("^notes:\\/\\/\\S+", "i") },
	file: { match: new RegExp("^file:\\/\\/\\/?\\S+", "i") },
	unc: { match: new RegExp("^\\\\\\\\\\S+", "i") },
	ftp: {
		match: new RegExp("^ftp:\\/\\/" + domainPortWithUrlRegEx + "|ftp\\." + domainPortWithUrlRegEx, "i"),
		normalizeUrl: function(url) {
			return new RegExp("^ftp:\\/\\/", "i").test(url) ? url : "ftp://" + url;
		}
	},
	news: { match: new RegExp("^news:(\\/\\/)?" + domainPortWithUrlRegEx, "i") },
	telnet: { match: new RegExp("^telnet:(\\/\\/)?" + domainPortWithUrlRegEx, "i") },
	gopher: { match: new RegExp("^gopher:\\/\\/" + domainPortWithUrlRegEx, "i") },
	wais: { match: new RegExp("^wais:(\\/\\/)?" + domainPortWithUrlRegEx, "i") }
};
function matchLink(url) {
	var e_1, _a$5;
	if (url) try {
		for (var _b$1 = __values(getObjectKeys(linkMatchRules)), _c$1 = _b$1.next(); !_c$1.done; _c$1 = _b$1.next()) {
			var schema = _c$1.value;
			var rule = linkMatchRules[schema];
			var matches = url.match(rule.match);
			if (matches && matches[0] == url && (!rule.except || !rule.except.test(url))) return {
				scheme: schema,
				originalUrl: url,
				normalizedUrl: rule.normalizeUrl ? rule.normalizeUrl(url) : url
			};
		}
	} catch (e_1_1) {
		e_1 = { error: e_1_1 };
	} finally {
		try {
			if (_c$1 && !_c$1.done && (_a$5 = _b$1.return)) _a$5.call(_b$1);
		} finally {
			if (e_1) throw e_1.error;
		}
	}
	return null;
}
function formatInsertPointWithContentModel(editor, insertPoint, callback, options) {
	var bundle = { input: insertPoint };
	editor.formatContentModel(function(model, context) {
		callback(model, context, bundle.result);
		if (bundle === null || bundle === void 0 ? void 0 : bundle.result) {
			var _a$5 = bundle.result, paragraph = _a$5.paragraph, marker = _a$5.marker;
			var index$1 = paragraph.segments.indexOf(marker);
			if (index$1 >= 0) mutateBlock(paragraph).segments.splice(index$1, 1);
		}
		return true;
	}, options, {
		processorOverride: {
			child: getShadowChildProcessor(bundle),
			textWithSelection: getShadowTextProcessor(bundle)
		},
		tryGetFromCache: false
	});
}
function getShadowChildProcessor(bundle) {
	return function(group, parent, context) {
		var contextWithPath = context;
		contextWithPath.path = contextWithPath.path || [];
		var shouldShiftPath = false;
		if (contextWithPath.path[0] != group) {
			contextWithPath.path.unshift(group);
			shouldShiftPath = true;
		}
		var offsets = getShadowSelectionOffsets(context, bundle, parent);
		var index$1 = 0;
		for (var child$1 = parent.firstChild; child$1; child$1 = child$1.nextSibling) {
			handleElementShadowSelection(bundle, index$1, context, group, offsets, parent);
			processChildNode(group, child$1, context);
			index$1++;
		}
		handleElementShadowSelection(bundle, index$1, context, group, offsets, parent);
		if (shouldShiftPath) contextWithPath.path.shift();
	};
}
function handleElementShadowSelection(bundle, index$1, context, group, offsets, container) {
	if (index$1 == offsets[2] && (index$1 <= offsets[0] || offsets[0] < 0) && (index$1 < offsets[1] || offsets[1] < 0)) {
		addSelectionMarker(group, context, container, index$1, bundle);
		offsets[2] = -1;
	}
	if (index$1 == offsets[0]) {
		context.isInSelection = true;
		addSelectionMarker(group, context, container, index$1);
	}
	if (index$1 == offsets[2] && (index$1 < offsets[1] || offsets[1] < 0)) {
		addSelectionMarker(group, context, container, index$1, bundle);
		offsets[2] = -1;
	}
	if (index$1 == offsets[1]) {
		addSelectionMarker(group, context, container, index$1);
		context.isInSelection = false;
	}
	if (index$1 == offsets[2]) addSelectionMarker(group, context, container, index$1, bundle);
}
var getShadowTextProcessor = function(bundle) {
	return function(group, textNode, context) {
		var txt = textNode.nodeValue || "";
		var offsets = getShadowSelectionOffsets(context, bundle, textNode);
		var _a$5 = __read(offsets, 3), start = _a$5[0], end = _a$5[1], shadow = _a$5[2];
		var handleTextSelection = function(subtract, originalOffset, bundle$1) {
			addTextSegment(group, txt.substring(0, subtract), context);
			addSelectionMarker(group, context, textNode, originalOffset, bundle$1);
			offsets[0] -= subtract;
			offsets[1] -= subtract;
			offsets[2] = bundle$1 ? -1 : offsets[2] - subtract;
			txt = txt.substring(subtract);
		};
		if (offsets[2] >= 0 && (offsets[2] <= offsets[0] || offsets[0] < 0) && (offsets[2] < offsets[1] || offsets[1] < 0)) handleTextSelection(offsets[2], shadow, bundle);
		if (offsets[0] >= 0) {
			handleTextSelection(offsets[0], start);
			context.isInSelection = true;
		}
		if (offsets[2] >= 0 && offsets[2] > offsets[0] && (offsets[2] < offsets[1] || offsets[1] < 0)) handleTextSelection(offsets[2], shadow, bundle);
		if (offsets[1] >= 0) {
			handleTextSelection(offsets[1], end);
			context.isInSelection = false;
		}
		if (offsets[2] >= 0 && offsets[2] >= offsets[1]) handleTextSelection(offsets[2], shadow, bundle);
		addTextSegment(group, txt, context);
	};
};
function addSelectionMarker(group, context, container, offset, bundle) {
	var marker = buildSelectionMarker(group, context, container, offset);
	marker.isSelected = !bundle;
	var para = addSegment(group, marker, context.blockFormat, marker.format);
	if (bundle && context.path) bundle.result = {
		path: __spreadArray([], __read(context.path), false),
		paragraph: para,
		marker
	};
}
function getShadowSelectionOffsets(context, bundle, currentContainer) {
	var _a$5 = __read(getRegularSelectionOffsets(context, currentContainer), 2);
	return [
		_a$5[0],
		_a$5[1],
		bundle.input.node == currentContainer ? bundle.input.offset : -1
	];
}
function formatTextSegmentBeforeSelectionMarker(editor, callback, options) {
	var result = false;
	editor.formatContentModel(function(model, context) {
		var selectedSegmentsAndParagraphs = getSelectedSegmentsAndParagraphs(model, false);
		var rewrite = false;
		if (selectedSegmentsAndParagraphs.length > 0 && selectedSegmentsAndParagraphs[0][0].segmentType == "SelectionMarker" && selectedSegmentsAndParagraphs[0][1]) mutateSegment(selectedSegmentsAndParagraphs[0][1], selectedSegmentsAndParagraphs[0][0], function(marker, paragraph, markerIndex) {
			var previousSegment = paragraph.segments[markerIndex - 1];
			if (previousSegment && previousSegment.segmentType === "Text") {
				result = true;
				context.newPendingFormat = "preserve";
				rewrite = callback(model, previousSegment, paragraph, marker.format, context);
			}
		});
		return rewrite;
	}, options);
	return result;
}
var COMMON_REGEX = "[s]*[a-zA-Z0-9+][s]*";
var TELEPHONE_REGEX = "(T|t)el:" + COMMON_REGEX;
var MAILTO_REGEX = "(M|m)ailto:" + COMMON_REGEX;
function getLinkUrl(text, autoLinkOptions) {
	var _a$5;
	var _b$1 = autoLinkOptions !== null && autoLinkOptions !== void 0 ? autoLinkOptions : {}, autoLink = _b$1.autoLink, autoMailto = _b$1.autoMailto, autoTel = _b$1.autoTel;
	var linkMatch = autoLink ? (_a$5 = matchLink(text)) === null || _a$5 === void 0 ? void 0 : _a$5.normalizedUrl : void 0;
	var telMatch = autoTel ? matchTel(text) : void 0;
	var mailtoMatch = autoMailto ? matchMailTo(text) : void 0;
	return linkMatch || telMatch || mailtoMatch;
}
function matchTel(text) {
	return text.match(TELEPHONE_REGEX) ? text.toLocaleLowerCase() : void 0;
}
function matchMailTo(text) {
	return text.match(MAILTO_REGEX) ? text.toLocaleLowerCase() : void 0;
}
function promoteLink(segment, paragraph, autoLinkOptions) {
	if (segment.link) return null;
	var link = segment.text.split(" ").pop();
	var url = link === null || link === void 0 ? void 0 : link.trim();
	var linkUrl = void 0;
	if (url && link && (linkUrl = getLinkUrl(url, autoLinkOptions))) {
		var linkSegment = splitTextSegment(segment, paragraph, segment.text.length - link.trimLeft().length, segment.text.trimRight().length);
		linkSegment.link = {
			format: {
				href: linkUrl,
				underline: true
			},
			dataset: {}
		};
		return linkSegment;
	}
	return null;
}
function queryContentModelBlocks(group, type, filter, findFirstOnly, shouldExpandEntity) {
	var e_1, _a$5, e_2, _b$1;
	var elements = [];
	for (var i$1 = 0; i$1 < group.blocks.length; i$1++) {
		if (findFirstOnly && elements.length > 0) return elements;
		var block = group.blocks[i$1];
		switch (block.blockType) {
			case "Paragraph":
			case "Divider":
			case "Entity":
				if (isExpectedBlockType(block, type, filter)) elements.push(block);
				if (block.blockType == "Entity" && shouldExpandEntity) {
					var editorContext = shouldExpandEntity(block);
					if (editorContext) {
						var context = createDomToModelContext(editorContext);
						var results_1 = queryContentModelBlocks(domToContentModel(block.wrapper, context), type, filter, findFirstOnly, shouldExpandEntity);
						elements.push.apply(elements, __spreadArray([], __read(results_1), false));
					}
				}
				break;
			case "BlockGroup":
				if (isExpectedBlockType(block, type, filter)) elements.push(block);
				var results = queryContentModelBlocks(block, type, filter, findFirstOnly, shouldExpandEntity);
				elements.push.apply(elements, __spreadArray([], __read(results), false));
				break;
			case "Table":
				if (isExpectedBlockType(block, type, filter)) elements.push(block);
				try {
					for (var _c$1 = (e_1 = void 0, __values(block.rows)), _d$1 = _c$1.next(); !_d$1.done; _d$1 = _c$1.next()) {
						var row = _d$1.value;
						try {
							for (var _e$1 = (e_2 = void 0, __values(row.cells)), _f$1 = _e$1.next(); !_f$1.done; _f$1 = _e$1.next()) {
								var cell = _f$1.value;
								var results_2 = queryContentModelBlocks(cell, type, filter, findFirstOnly, shouldExpandEntity);
								elements.push.apply(elements, __spreadArray([], __read(results_2), false));
							}
						} catch (e_2_1) {
							e_2 = { error: e_2_1 };
						} finally {
							try {
								if (_f$1 && !_f$1.done && (_b$1 = _e$1.return)) _b$1.call(_e$1);
							} finally {
								if (e_2) throw e_2.error;
							}
						}
					}
				} catch (e_1_1) {
					e_1 = { error: e_1_1 };
				} finally {
					try {
						if (_d$1 && !_d$1.done && (_a$5 = _c$1.return)) _a$5.call(_c$1);
					} finally {
						if (e_1) throw e_1.error;
					}
				}
				break;
		}
	}
	return elements;
}
function isExpectedBlockType(block, type, filter) {
	return isBlockType(block, type) && (!filter || filter(block));
}
function isBlockType(block, type) {
	return block.blockType == type;
}
var INSERTER_COLOR = "#4A4A4A";
var INSERTER_COLOR_DARK_MODE = "white";
var INSERTER_SIDE_LENGTH = 12;
var INSERTER_BORDER_SIZE = 1;
var HORIZONTAL_INSERTER_ID = "horizontalInserter";
var VERTICAL_INSERTER_ID = "verticalInserter";
function createTableInserter(editor, td, table, isRTL, isHorizontal, onBeforeInsert, onAfterInserted, anchorContainer, onTableEditorCreated) {
	var tdRect = normalizeRect(td.getBoundingClientRect());
	var viewPort = editor.getVisibleViewport();
	var tableRect = table && viewPort ? getIntersectedRect([table], [viewPort]) : null;
	if (tdRect && tableRect) {
		var document_1 = td.ownerDocument;
		var div = createElement(getInsertElementData(isHorizontal, editor.isDarkMode(), isRTL, editor.getDOMHelper().getDomStyle("backgroundColor") || "white"), document_1);
		if (isHorizontal) {
			div.id = HORIZONTAL_INSERTER_ID;
			div.style.left = (isRTL ? tableRect.right : tableRect.left - (INSERTER_SIDE_LENGTH - 1 + 2 * INSERTER_BORDER_SIZE)) + "px";
			div.style.top = tdRect.bottom - 8 + "px";
			div.firstChild.style.width = tableRect.right - tableRect.left + "px";
		} else {
			div.id = VERTICAL_INSERTER_ID;
			div.style.left = (isRTL ? tdRect.left - 8 : tdRect.right - 8) + "px";
			div.style.top = tableRect.top - (INSERTER_SIDE_LENGTH - 1 + 2 * INSERTER_BORDER_SIZE) + "px";
			div.firstChild.style.height = tableRect.bottom - tableRect.top + "px";
		}
		(anchorContainer || document_1.body).appendChild(div);
		return {
			div,
			featureHandler: new TableInsertHandler(div, td, table, isHorizontal, editor, onBeforeInsert, onAfterInserted, onTableEditorCreated),
			node: td
		};
	}
	return null;
}
var TableInsertHandler = function() {
	function TableInsertHandler$1(div, td, table, isHorizontal, editor, onBeforeInsert, onAfterInsert, onTableEditorCreated) {
		var _this = this;
		this.div = div;
		this.td = td;
		this.table = table;
		this.isHorizontal = isHorizontal;
		this.editor = editor;
		this.onBeforeInsert = onBeforeInsert;
		this.onAfterInsert = onAfterInsert;
		this.insertTd = function() {
			var columnIndex = _this.td.cellIndex;
			var row = _this.td.parentElement && isElementOfType(_this.td.parentElement, "tr") ? _this.td.parentElement : void 0;
			var rowIndex = row && row.rowIndex;
			if ((row === null || row === void 0 ? void 0 : row.cells) == void 0 || rowIndex == void 0) return;
			_this.onBeforeInsert();
			formatTableWithContentModel(_this.editor, "editTablePlugin", function(tableModel) {
				_this.isHorizontal ? insertTableRow(tableModel, "insertBelow") : insertTableColumn(tableModel, "insertRight");
			}, {
				type: "table",
				firstColumn: columnIndex,
				firstRow: rowIndex,
				lastColumn: columnIndex,
				lastRow: rowIndex,
				table: _this.table
			});
			_this.onAfterInsert();
		};
		this.div.addEventListener("click", this.insertTd);
		this.disposer = onTableEditorCreated === null || onTableEditorCreated === void 0 ? void 0 : onTableEditorCreated(isHorizontal ? "HorizontalTableInserter" : "VerticalTableInserter", div);
	}
	TableInsertHandler$1.prototype.dispose = function() {
		var _a$5;
		this.div.removeEventListener("click", this.insertTd);
		(_a$5 = this.disposer) === null || _a$5 === void 0 || _a$5.call(this);
		this.disposer = void 0;
	};
	return TableInsertHandler$1;
}();
function getInsertElementData(isHorizontal, isDark, isRTL, backgroundColor) {
	var inserterColor = isDark ? INSERTER_COLOR_DARK_MODE : INSERTER_COLOR;
	var outerDivStyle = "position: fixed; width: " + INSERTER_SIDE_LENGTH + "px; height: " + INSERTER_SIDE_LENGTH + "px; font-size: 16px; color: black; line-height: 8px; vertical-align: middle; text-align: center; cursor: pointer; border: solid " + INSERTER_BORDER_SIZE + "px " + inserterColor + "; border-radius: 50%; background-color: " + backgroundColor;
	var leftOrRight = isRTL ? "right" : "left";
	return {
		tag: "div",
		style: outerDivStyle,
		children: [{
			tag: "div",
			style: "position: absolute; box-sizing: border-box; background-color: " + backgroundColor + ";" + (isHorizontal ? leftOrRight + ": 12px; top: 5px; height: 3px; border-top: 1px solid " + inserterColor + "; border-bottom: 1px solid " + inserterColor + "; border-right: 1px solid " + inserterColor + "; border-left: 0px;" : "left: 5px; top: 12px; width: 3px; border-left: 1px solid " + inserterColor + "; border-right: 1px solid " + inserterColor + "; border-bottom: 1px solid " + inserterColor + "; border-top: 0px;")
		}, "+"]
	};
}
function getNodePositionFromEvent(editor, x$1, y$1) {
	var doc = editor.getDocument();
	var domHelper = editor.getDOMHelper();
	if ("caretPositionFromPoint" in doc) {
		var pos = doc.caretPositionFromPoint(x$1, y$1);
		if (pos && domHelper.isNodeInEditor(pos.offsetNode)) return {
			node: pos.offsetNode,
			offset: pos.offset
		};
	}
	if (doc.caretRangeFromPoint) {
		var range = doc.caretRangeFromPoint(x$1, y$1);
		if (range && domHelper.isNodeInEditor(range.startContainer)) return {
			node: range.startContainer,
			offset: range.startOffset
		};
	}
	if (doc.elementFromPoint) {
		var element = doc.elementFromPoint(x$1, y$1);
		if (element && domHelper.isNodeInEditor(element)) return {
			node: element,
			offset: 0
		};
	}
	return null;
}
var TABLE_MOVER_LENGTH = 12;
var TABLE_MOVER_ID = "_Table_Mover";
var TABLE_MOVER_STYLE_KEY = "_TableMoverCursorStyle";
function createTableMover(table, editor, isRTL, onFinishDragging, onStart, onEnd, contentDiv, anchorContainer, onTableEditorCreated, disableMovement) {
	var rect = normalizeRect(table.getBoundingClientRect());
	if (!isTableTopVisible(editor, rect, contentDiv)) return null;
	var zoomScale = editor.getDOMHelper().calculateZoomScale();
	var document$1 = table.ownerDocument;
	var div = createElement({
		tag: "div",
		style: "position: fixed; cursor: move; user-select: none; border: 1px solid #808080"
	}, document$1);
	div.id = TABLE_MOVER_ID;
	div.style.width = TABLE_MOVER_LENGTH + "px";
	div.style.height = TABLE_MOVER_LENGTH + "px";
	(anchorContainer || document$1.body).appendChild(div);
	var context = {
		table,
		zoomScale,
		rect,
		isRTL,
		editor,
		div,
		onFinishDragging,
		onStart,
		onEnd,
		disableMovement
	};
	setDivPosition$1(context, div);
	return {
		node: table,
		div,
		featureHandler: new TableMoverFeature(div, context, function() {}, disableMovement ? { onDragEnd: onDragEnd$1 } : {
			onDragStart: onDragStart$1,
			onDragging: onDragging$1,
			onDragEnd: onDragEnd$1
		}, context.zoomScale, onTableEditorCreated, editor.getEnvironment().isMobileOrTablet)
	};
}
var TableMoverFeature = function(_super) {
	__extends(TableMoverFeature$1, _super);
	function TableMoverFeature$1(div, context, onSubmit, handler, zoomScale, onTableEditorCreated, forceMobile) {
		var _this = _super.call(this, div, context, onSubmit, handler, zoomScale, forceMobile) || this;
		_this.disposer = onTableEditorCreated === null || onTableEditorCreated === void 0 ? void 0 : onTableEditorCreated("TableMover", div);
		return _this;
	}
	TableMoverFeature$1.prototype.dispose = function() {
		var _a$5;
		(_a$5 = this.disposer) === null || _a$5 === void 0 || _a$5.call(this);
		this.disposer = void 0;
		_super.prototype.dispose.call(this);
	};
	return TableMoverFeature$1;
}(DragAndDropHelper);
function setDivPosition$1(context, trigger) {
	var rect = context.rect;
	if (rect) {
		trigger.style.top = rect.top - TABLE_MOVER_LENGTH + "px";
		trigger.style.left = rect.left - TABLE_MOVER_LENGTH - 2 + "px";
	}
}
function isTableTopVisible(editor, rect, contentDiv) {
	var visibleViewport = editor.getVisibleViewport();
	if (isNodeOfType(contentDiv, "ELEMENT_NODE") && visibleViewport && rect) {
		var containerRect = normalizeRect(contentDiv.getBoundingClientRect());
		return !!containerRect && containerRect.top <= rect.top && visibleViewport.top <= rect.top;
	}
	return true;
}
function setTableMoverCursor(editor, state$1, type) {
	var _a$5;
	editor === null || editor === void 0 || editor.setEditorStyle(TABLE_MOVER_STYLE_KEY, state$1 ? (_a$5 = "cursor: " + type) !== null && _a$5 !== void 0 ? _a$5 : "move" : null);
}
function onDragStart$1(context) {
	var _a$5;
	context.onStart();
	var editor = context.editor, table = context.table, div = context.div;
	setTableMoverCursor(editor, true, "move");
	var trect = table.getBoundingClientRect();
	var tableRect = createElement({
		tag: "div",
		style: "position: fixed; user-select: none; border: 1px solid #808080"
	}, document);
	tableRect.style.width = trect.width + "px";
	tableRect.style.height = trect.height + "px";
	tableRect.style.top = trect.top + "px";
	tableRect.style.left = trect.left + "px";
	(_a$5 = div.parentNode) === null || _a$5 === void 0 || _a$5.appendChild(tableRect);
	var initialSelection = editor.getDOMSelection();
	return {
		cmTable: getCMTableFromTable(editor, table),
		initialSelection,
		tableRect
	};
}
function onDragging$1(context, event$1, initValue) {
	var tableRect = initValue.tableRect;
	var editor = context.editor;
	tableRect.style.top = event$1.clientY + TABLE_MOVER_LENGTH + "px";
	tableRect.style.left = event$1.clientX + TABLE_MOVER_LENGTH + "px";
	var pos = getNodePositionFromEvent(editor, event$1.clientX, event$1.clientY);
	if (pos) {
		var range = editor.getDocument().createRange();
		range.setStart(pos.node, pos.offset);
		range.collapse(true);
		editor.setDOMSelection({
			type: "range",
			range,
			isReverted: false
		});
		return true;
	}
	return false;
}
function onDragEnd$1(context, event$1, initValue) {
	var _a$5, _b$1;
	var editor = context.editor, table = context.table, selectWholeTable = context.onFinishDragging, disableMovement = context.disableMovement;
	var element = event$1.target;
	initValue === null || initValue === void 0 || initValue.tableRect.remove();
	setTableMoverCursor(editor, false);
	if (element == context.div) {
		selectWholeTable(table);
		context.onEnd(false);
		return true;
	} else {
		if (table.contains(element) || !editor.getDOMHelper().isNodeInEditor(element) || disableMovement) {
			editor.setDOMSelection((_a$5 = initValue === null || initValue === void 0 ? void 0 : initValue.initialSelection) !== null && _a$5 !== void 0 ? _a$5 : null);
			context.onEnd(true);
			return false;
		}
		var insertionSuccess_1 = false;
		var insertPosition = getNodePositionFromEvent(editor, event$1.clientX, event$1.clientY);
		if (insertPosition) formatInsertPointWithContentModel(editor, insertPosition, function(model, context$1, ip) {
			var _a$6;
			var _b$2 = __read(getFirstSelectedTable(model), 2), oldTable = _b$2[0], path = _b$2[1];
			if (oldTable) {
				var index$1 = path[0].blocks.indexOf(oldTable);
				mutateBlock(path[0]).blocks.splice(index$1, 1);
			}
			if (ip && (initValue === null || initValue === void 0 ? void 0 : initValue.cmTable)) {
				var doc = createContentModelDocument();
				doc.blocks.push(oldTable !== null && oldTable !== void 0 ? oldTable : mutateBlock(initValue.cmTable));
				insertionSuccess_1 = !!mergeModel(model, cloneModel(doc), context$1, {
					mergeFormat: "none",
					insertPosition: ip
				});
				if (insertionSuccess_1) {
					var finalTable = (_a$6 = getFirstSelectedTable(model)[0]) !== null && _a$6 !== void 0 ? _a$6 : initValue.cmTable;
					if (finalTable) {
						var firstCell = finalTable.rows[0].cells[0];
						var markerParagraph = firstCell === null || firstCell === void 0 ? void 0 : firstCell.blocks[0];
						if ((markerParagraph === null || markerParagraph === void 0 ? void 0 : markerParagraph.blockType) == "Paragraph") {
							var marker = createSelectionMarker(model.format);
							mutateBlock(markerParagraph).segments.unshift(marker);
							setParagraphNotImplicit(markerParagraph);
							setSelection(model, marker);
						}
					}
				}
				return insertionSuccess_1;
			}
		}, {
			selectionOverride: {
				type: "table",
				firstColumn: 0,
				firstRow: 0,
				lastColumn: 0,
				lastRow: 0,
				table
			},
			apiName: "TableMover"
		});
		else editor.setDOMSelection((_b$1 = initValue === null || initValue === void 0 ? void 0 : initValue.initialSelection) !== null && _b$1 !== void 0 ? _b$1 : null);
		context.onEnd(true);
		return insertionSuccess_1;
	}
}
var TABLE_RESIZER_LENGTH$1 = 12;
var TABLE_RESIZER_ID = "_Table_Resizer";
function createTableResizer(table, editor, isRTL, onStart, onEnd, contentDiv, anchorContainer, onTableEditorCreated) {
	if (!isTableBottomVisible(editor, normalizeRect(table.getBoundingClientRect()), contentDiv)) return null;
	var document$1 = table.ownerDocument;
	var zoomScale = editor.getDOMHelper().calculateZoomScale();
	var div = createElement({
		tag: "div",
		style: "position: fixed; cursor: " + (isRTL ? "ne" : "nw") + "-resize; user-select: none; border: 1px solid #808080"
	}, document$1);
	div.id = TABLE_RESIZER_ID;
	div.style.width = TABLE_RESIZER_LENGTH$1 + "px";
	div.style.height = TABLE_RESIZER_LENGTH$1 + "px";
	(anchorContainer || document$1.body).appendChild(div);
	var context = {
		isRTL,
		table,
		zoomScale,
		onStart,
		onEnd,
		div,
		editor,
		contentDiv
	};
	setDivPosition(context, div);
	return {
		node: table,
		div,
		featureHandler: new TableResizer(div, context, hideResizer, {
			onDragStart,
			onDragging,
			onDragEnd
		}, zoomScale, editor.getEnvironment().isMobileOrTablet, onTableEditorCreated)
	};
}
var TableResizer = function(_super) {
	__extends(TableResizer$1, _super);
	function TableResizer$1(trigger, context, onSubmit, handler, zoomScale, forceMobile, onTableEditorCreated) {
		var _this = _super.call(this, trigger, context, onSubmit, handler, zoomScale, forceMobile) || this;
		_this.disposer = onTableEditorCreated === null || onTableEditorCreated === void 0 ? void 0 : onTableEditorCreated("TableResizer", trigger);
		return _this;
	}
	TableResizer$1.prototype.dispose = function() {
		var _a$5;
		(_a$5 = this.disposer) === null || _a$5 === void 0 || _a$5.call(this);
		this.disposer = void 0;
		_super.prototype.dispose.call(this);
	};
	return TableResizer$1;
}(DragAndDropHelper);
function onDragStart(context, event$1) {
	context.onStart();
	var editor = context.editor, table = context.table;
	var cmTable = getCMTableFromTable(editor, table);
	var heights = [];
	cmTable === null || cmTable === void 0 || cmTable.rows.forEach(function(row) {
		heights.push(row.height);
	});
	var widths = [];
	cmTable === null || cmTable === void 0 || cmTable.widths.forEach(function(width) {
		widths.push(width);
	});
	return {
		originalRect: table.getBoundingClientRect(),
		cmTable,
		originalHeights: heights !== null && heights !== void 0 ? heights : [],
		originalWidths: widths !== null && widths !== void 0 ? widths : []
	};
}
function onDragging(context, event$1, initValue, deltaX, deltaY) {
	var _a$5, _b$1;
	var isRTL = context.isRTL, zoomScale = context.zoomScale, table = context.table;
	var originalRect = initValue.originalRect, originalHeights = initValue.originalHeights, originalWidths = initValue.originalWidths, cmTable = initValue.cmTable;
	var ratioX = 1 + deltaX / originalRect.width * zoomScale * (isRTL ? -1 : 1);
	var ratioY = 1 + deltaY / originalRect.height * zoomScale;
	var shouldResizeX = Math.abs(ratioX - 1) > .001;
	var shouldResizeY = Math.abs(ratioY - 1) > .001;
	table.style.setProperty("width", null);
	table.style.setProperty("height", null);
	if (cmTable && cmTable.rows && (shouldResizeX || shouldResizeY)) {
		var mutableTable = mutateBlock(cmTable);
		for (var i$1 = 0; i$1 < cmTable.rows.length; i$1++) for (var j$1 = 0; j$1 < cmTable.rows[i$1].cells.length; j$1++) if (cmTable.rows[i$1].cells[j$1]) {
			if (shouldResizeX && i$1 == 0) mutableTable.widths[j$1] = ((_a$5 = originalWidths[j$1]) !== null && _a$5 !== void 0 ? _a$5 : 0) * ratioX;
			if (shouldResizeY && j$1 == 0) mutableTable.rows[i$1].height = ((_b$1 = originalHeights[i$1]) !== null && _b$1 !== void 0 ? _b$1 : 0) * ratioY;
		}
		for (var row = 0; row < table.rows.length; row++) {
			var tableRow = table.rows[row];
			if (tableRow.cells.length == 0) continue;
			var newHeight = Math.max(cmTable.rows[row].height, 22);
			for (var col = 0; col < tableRow.cells.length; col++) {
				var td = tableRow.cells[col];
				var newWidth = Math.max(cmTable.widths[col], 22);
				td.style.width = newWidth + "px";
				td.style.height = newHeight + "px";
				td.style.boxSizing = "border-box";
			}
		}
		return true;
	} else return false;
}
function onDragEnd(context, event$1, initValue) {
	if (context.editor.isDisposed()) return false;
	if (isTableBottomVisible(context.editor, normalizeRect(context.table.getBoundingClientRect()), context.contentDiv)) {
		context.div.style.visibility = "visible";
		setDivPosition(context, context.div);
	}
	context.onEnd();
	return false;
}
function setDivPosition(context, trigger) {
	var table = context.table, isRTL = context.isRTL;
	var rect = normalizeRect(table.getBoundingClientRect());
	if (rect) {
		trigger.style.top = rect.bottom + "px";
		trigger.style.left = isRTL ? rect.left - TABLE_RESIZER_LENGTH$1 - 2 + "px" : rect.right + "px";
	}
}
function hideResizer(context, trigger) {
	trigger.style.visibility = "hidden";
}
function isTableBottomVisible(editor, rect, contentDiv) {
	var visibleViewport = editor.getVisibleViewport();
	if (isNodeOfType(contentDiv, "ELEMENT_NODE") && visibleViewport && rect) {
		var containerRect = normalizeRect(contentDiv.getBoundingClientRect());
		return !!containerRect && containerRect.bottom >= rect.bottom && visibleViewport.bottom >= rect.bottom;
	}
	return true;
}
function disposeTableEditFeature(feature) {
	var _a$5, _b$1, _c$1;
	if (feature) {
		(_a$5 = feature.featureHandler) === null || _a$5 === void 0 || _a$5.dispose();
		feature.featureHandler = null;
		(_c$1 = (_b$1 = feature.div) === null || _b$1 === void 0 ? void 0 : _b$1.parentNode) === null || _c$1 === void 0 || _c$1.removeChild(feature.div);
		feature.div = null;
	}
}
var INSERTER_HOVER_OFFSET = 6;
var TOP_OR_SIDE;
(function(TOP_OR_SIDE$1) {
	TOP_OR_SIDE$1[TOP_OR_SIDE$1["top"] = 0] = "top";
	TOP_OR_SIDE$1[TOP_OR_SIDE$1["side"] = 1] = "side";
})(TOP_OR_SIDE || (TOP_OR_SIDE = {}));
var TableEditor = function() {
	function TableEditor$1(editor, table, logicalRoot, onChanged, anchorContainer, contentDiv, onTableEditorCreated, disableFeatures) {
		var _this = this;
		var _a$5;
		this.editor = editor;
		this.table = table;
		this.logicalRoot = logicalRoot;
		this.onChanged = onChanged;
		this.anchorContainer = anchorContainer;
		this.contentDiv = contentDiv;
		this.onTableEditorCreated = onTableEditorCreated;
		this.disableFeatures = disableFeatures;
		this.horizontalInserter = null;
		this.verticalInserter = null;
		this.horizontalResizer = null;
		this.verticalResizer = null;
		this.tableResizer = null;
		this.tableMover = null;
		this.range = null;
		this.onEditorCreated = function(featureType, element) {
			var _a$6;
			var disposer = (_a$6 = _this.onTableEditorCreated) === null || _a$6 === void 0 ? void 0 : _a$6.call(_this, featureType, element);
			var onMouseOut = element && _this.getOnMouseOut(element);
			if (onMouseOut) element.addEventListener("mouseout", onMouseOut);
			return function() {
				disposer === null || disposer === void 0 || disposer();
				if (onMouseOut) element.removeEventListener("mouseout", onMouseOut);
			};
		};
		this.onFinishEditing = function() {
			_this.editor.focus();
			if (_this.range) {
				_this.editor.setDOMSelection({
					type: "range",
					range: _this.range,
					isReverted: false
				});
				_this.range = null;
			}
			_this.editor.takeSnapshot();
			_this.onChanged();
			_this.isCurrentlyEditing = false;
			return false;
		};
		this.onStartTableResize = function() {
			_this.isCurrentlyEditing = true;
			_this.onStartResize();
		};
		this.onStartCellResize = function() {
			_this.isCurrentlyEditing = true;
			_this.disposeTableResizer();
			_this.onStartResize();
		};
		this.onStartTableMove = function() {
			_this.onBeforeEditTable();
			_this.isCurrentlyEditing = true;
			_this.disposeTableResizer();
			_this.disposeTableInserter();
			_this.disposeCellResizers();
		};
		this.onEndTableMove = function(disposeHandler) {
			if (disposeHandler) _this.disposeTableMover();
			return _this.onFinishEditing();
		};
		this.onBeforeEditTable = function() {
			_this.editor.setLogicalRoot(_this.logicalRoot);
		};
		this.onAfterInsert = function() {
			_this.disposeTableResizer();
			_this.onFinishEditing();
		};
		this.onSelect = function(table$1) {
			var _a$6, _b$1;
			_this.editor.focus();
			if (table$1) {
				var parsedTable = parseTableCells(table$1);
				var selection = {
					table: table$1,
					firstRow: 0,
					firstColumn: 0,
					lastRow: parsedTable.length - 1,
					lastColumn: ((_b$1 = (_a$6 = parsedTable[0]) === null || _a$6 === void 0 ? void 0 : _a$6.length) !== null && _b$1 !== void 0 ? _b$1 : 0) - 1,
					type: "table"
				};
				_this.editor.setDOMSelection(selection);
			}
		};
		this.getOnMouseOut = function(feature) {
			return function(ev) {
				if (feature && ev.relatedTarget != feature && isNodeOfType(_this.contentDiv, "ELEMENT_NODE") && isNodeOfType(ev.relatedTarget, "ELEMENT_NODE") && !(_this.contentDiv == ev.relatedTarget) && !_this.isEditing()) _this.dispose();
			};
		};
		this.isRTL = ((_a$5 = editor.getDocument().defaultView) === null || _a$5 === void 0 ? void 0 : _a$5.getComputedStyle(table).direction) == "rtl";
		this.setEditorFeatures();
		this.isCurrentlyEditing = false;
	}
	TableEditor$1.prototype.dispose = function() {
		this.disposeTableResizer();
		this.disposeCellResizers();
		this.disposeTableInserter();
		this.disposeTableMover();
	};
	TableEditor$1.prototype.isEditing = function() {
		return this.isCurrentlyEditing;
	};
	TableEditor$1.prototype.isOwnedElement = function(node) {
		return [
			this.tableResizer,
			this.tableMover,
			this.horizontalInserter,
			this.verticalInserter,
			this.horizontalResizer,
			this.verticalResizer
		].filter(function(feature) {
			return !!(feature === null || feature === void 0 ? void 0 : feature.div);
		}).some(function(feature) {
			return (feature === null || feature === void 0 ? void 0 : feature.div) == node;
		});
	};
	TableEditor$1.prototype.onMouseMove = function(x$1, y$1) {
		var _a$5;
		var tableRect = normalizeRect(this.table.getBoundingClientRect());
		if (!tableRect) return;
		var topOrSide = y$1 <= tableRect.top + INSERTER_HOVER_OFFSET ? 0 : this.isRTL ? x$1 >= tableRect.right - INSERTER_HOVER_OFFSET ? 1 : void 0 : x$1 <= tableRect.left + INSERTER_HOVER_OFFSET ? 1 : void 0;
		var topOrSideBinary = topOrSide ? 1 : 0;
		for (var i$1 = 0; i$1 < this.table.rows.length; i$1++) {
			var tr = this.table.rows[i$1];
			var j$1 = 0;
			for (; j$1 < tr.cells.length; j$1++) {
				var td = tr.cells[j$1];
				var tdRect = normalizeRect(td.getBoundingClientRect());
				if (!tdRect || !tableRect) continue;
				var lessThanBottom = y$1 <= tdRect.bottom;
				var lessThanRight = this.isRTL ? x$1 <= tdRect.right + INSERTER_HOVER_OFFSET * topOrSideBinary : x$1 <= tdRect.right;
				var moreThanLeft = this.isRTL ? x$1 >= tdRect.left : x$1 >= tdRect.left - INSERTER_HOVER_OFFSET * topOrSideBinary;
				if (lessThanBottom && lessThanRight && moreThanLeft) {
					if (i$1 === 0 && topOrSide == 0) {
						var center = (tdRect.left + tdRect.right) / 2;
						var isOnRightHalf = this.isRTL ? x$1 < center : x$1 > center;
						!this.isFeatureDisabled("VerticalTableInserter") && this.setInserterTd(isOnRightHalf ? td : tr.cells[j$1 - 1], false);
					} else if (j$1 === 0 && topOrSide == 1) {
						var tdAbove = (_a$5 = this.table.rows[i$1 - 1]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[0];
						var tdAboveRect = tdAbove ? normalizeRect(tdAbove.getBoundingClientRect()) : null;
						var isTdNotAboveMerged = !tdAboveRect ? null : this.isRTL ? tdAboveRect.right === tdRect.right : tdAboveRect.left === tdRect.left;
						!this.isFeatureDisabled("HorizontalTableInserter") && this.setInserterTd(y$1 < (tdRect.top + tdRect.bottom) / 2 && isTdNotAboveMerged ? tdAbove : td, true);
					} else this.setInserterTd(null);
					!this.isFeatureDisabled("CellResizer") && this.setResizingTd(td);
					break;
				}
			}
			if (j$1 < tr.cells.length) break;
		}
		this.setEditorFeatures();
	};
	TableEditor$1.prototype.setEditorFeatures = function() {
		var disableSelector = this.isFeatureDisabled("TableSelector");
		var disableMovement = this.isFeatureDisabled("TableMover");
		if (!this.tableMover && !(disableSelector && disableMovement)) this.tableMover = createTableMover(this.table, this.editor, this.isRTL, disableSelector ? function() {} : this.onSelect, this.onStartTableMove, this.onEndTableMove, this.contentDiv, this.anchorContainer, this.onEditorCreated, disableMovement);
		if (!this.tableResizer && !this.isFeatureDisabled("TableResizer")) this.tableResizer = createTableResizer(this.table, this.editor, this.isRTL, this.onStartTableResize, this.onFinishEditing, this.contentDiv, this.anchorContainer, this.onTableEditorCreated);
	};
	TableEditor$1.prototype.setResizingTd = function(td) {
		if (this.horizontalResizer && this.horizontalResizer.node != td) this.disposeCellResizers();
		if (!this.horizontalResizer && td) {
			this.horizontalResizer = createCellResizer(this.editor, td, this.table, this.isRTL, true, this.onStartCellResize, this.onFinishEditing, this.anchorContainer, this.onTableEditorCreated);
			this.verticalResizer = createCellResizer(this.editor, td, this.table, this.isRTL, false, this.onStartCellResize, this.onFinishEditing, this.anchorContainer, this.onTableEditorCreated);
		}
	};
	TableEditor$1.prototype.setInserterTd = function(td, isHorizontal) {
		var inserter = isHorizontal ? this.horizontalInserter : this.verticalInserter;
		if (td === null || inserter && inserter.node != td) this.disposeTableInserter();
		if (!this.horizontalInserter && !this.verticalInserter && td) {
			var newInserter = createTableInserter(this.editor, td, this.table, this.isRTL, !!isHorizontal, this.onBeforeEditTable, this.onAfterInsert, this.anchorContainer, this.onEditorCreated);
			if (isHorizontal) this.horizontalInserter = newInserter;
			else this.verticalInserter = newInserter;
		}
	};
	TableEditor$1.prototype.disposeTableResizer = function() {
		if (this.tableResizer) {
			disposeTableEditFeature(this.tableResizer);
			this.tableResizer = null;
		}
	};
	TableEditor$1.prototype.disposeTableInserter = function() {
		if (this.horizontalInserter) {
			disposeTableEditFeature(this.horizontalInserter);
			this.horizontalInserter = null;
		}
		if (this.verticalInserter) {
			disposeTableEditFeature(this.verticalInserter);
			this.verticalInserter = null;
		}
	};
	TableEditor$1.prototype.disposeCellResizers = function() {
		if (this.horizontalResizer) {
			disposeTableEditFeature(this.horizontalResizer);
			this.horizontalResizer = null;
		}
		if (this.verticalResizer) {
			disposeTableEditFeature(this.verticalResizer);
			this.verticalResizer = null;
		}
	};
	TableEditor$1.prototype.disposeTableMover = function() {
		if (this.tableMover) {
			disposeTableEditFeature(this.tableMover);
			this.tableMover = null;
		}
	};
	TableEditor$1.prototype.onStartResize = function() {
		this.onBeforeEditTable();
		this.isCurrentlyEditing = true;
		var range = this.editor.getDOMSelection();
		if (range && range.type == "range") this.range = range.range;
		this.editor.takeSnapshot();
	};
	TableEditor$1.prototype.isFeatureDisabled = function(feature) {
		var _a$5;
		return (_a$5 = this.disableFeatures) === null || _a$5 === void 0 ? void 0 : _a$5.includes(feature);
	};
	return TableEditor$1;
}();
var TABLE_RESIZER_LENGTH = 12;
var TableEditPlugin = function() {
	function TableEditPlugin$1(anchorContainerSelector, onTableEditorCreated, disableFeatures, tableSelector) {
		var _this = this;
		if (tableSelector === void 0) tableSelector = defaultTableSelector;
		this.anchorContainerSelector = anchorContainerSelector;
		this.onTableEditorCreated = onTableEditorCreated;
		this.disableFeatures = disableFeatures;
		this.tableSelector = tableSelector;
		this.editor = null;
		this.onMouseMoveDisposer = null;
		this.tableRectMap = null;
		this.tableEditor = null;
		this.onMouseOut = function(_a$5) {
			var relatedTarget = _a$5.relatedTarget, currentTarget = _a$5.currentTarget;
			var relatedTargetNode = relatedTarget;
			var currentTargetNode = currentTarget;
			if (isNodeOfType(relatedTargetNode, "ELEMENT_NODE") && isNodeOfType(currentTargetNode, "ELEMENT_NODE") && _this.tableEditor && !_this.tableEditor.isOwnedElement(relatedTargetNode) && !currentTargetNode.contains(relatedTargetNode)) _this.setTableEditor(null);
		};
		this.onMouseMove = function(event$1) {
			var _a$5;
			var e$1 = event$1;
			if (e$1.buttons > 0 || !_this.editor) return;
			_this.ensureTableRects();
			var editorWindow = _this.editor.getDocument().defaultView || window;
			var x$1 = e$1.pageX - editorWindow.scrollX;
			var y$1 = e$1.pageY - editorWindow.scrollY;
			var currentTable = null;
			if (_this.tableRectMap) for (var i$1 = _this.tableRectMap.length - 1; i$1 >= 0; i$1--) {
				var entry = _this.tableRectMap[i$1];
				var rect = entry.rect;
				if (x$1 >= rect.left - TABLE_RESIZER_LENGTH && x$1 <= rect.right + TABLE_RESIZER_LENGTH && y$1 >= rect.top - TABLE_RESIZER_LENGTH && y$1 <= rect.bottom + TABLE_RESIZER_LENGTH) {
					currentTable = entry;
					break;
				}
			}
			_this.setTableEditor(currentTable, e$1);
			(_a$5 = _this.tableEditor) === null || _a$5 === void 0 || _a$5.onMouseMove(x$1, y$1);
		};
		this.invalidateTableRects = function() {
			_this.tableRectMap = null;
		};
	}
	TableEditPlugin$1.prototype.getName = function() {
		return "TableEdit";
	};
	TableEditPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		this.onMouseMoveDisposer = this.editor.attachDomEvent({ mousemove: { beforeDispatch: this.onMouseMove } });
		this.editor.getScrollContainer().addEventListener("mouseout", this.onMouseOut);
	};
	TableEditPlugin$1.prototype.dispose = function() {
		var _a$5, _b$1;
		var scrollContainer = (_a$5 = this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getScrollContainer();
		scrollContainer === null || scrollContainer === void 0 || scrollContainer.removeEventListener("mouseout", this.onMouseOut);
		(_b$1 = this.onMouseMoveDisposer) === null || _b$1 === void 0 || _b$1.call(this);
		this.invalidateTableRects();
		this.disposeTableEditor();
		this.editor = null;
		this.onMouseMoveDisposer = null;
		this.onTableEditorCreated = void 0;
	};
	TableEditPlugin$1.prototype.onPluginEvent = function(e$1) {
		switch (e$1.eventType) {
			case "input":
			case "contentChanged":
			case "scroll":
			case "zoomChanged":
				this.setTableEditor(null);
				this.invalidateTableRects();
				break;
		}
	};
	TableEditPlugin$1.prototype.setTableEditor = function(entry, event$1) {
		if (this.tableEditor && !this.tableEditor.isEditing() && (entry === null || entry === void 0 ? void 0 : entry.table) != this.tableEditor.table) this.disposeTableEditor();
		if (!this.tableEditor && entry && this.editor && entry.table.rows.length > 0) {
			var container = this.anchorContainerSelector ? this.editor.getDocument().querySelector(this.anchorContainerSelector) : void 0;
			this.tableEditor = new TableEditor(this.editor, entry.table, entry.logicalRoot, this.invalidateTableRects, isNodeOfType(container, "ELEMENT_NODE") ? container : void 0, event$1 === null || event$1 === void 0 ? void 0 : event$1.currentTarget, this.onTableEditorCreated, this.disableFeatures);
		}
	};
	TableEditPlugin$1.prototype.disposeTableEditor = function() {
		var _a$5;
		(_a$5 = this.tableEditor) === null || _a$5 === void 0 || _a$5.dispose();
		this.tableEditor = null;
	};
	TableEditPlugin$1.prototype.ensureTableRects = function() {
		var _this = this;
		if (!this.tableRectMap && this.editor) {
			this.tableRectMap = [];
			this.tableSelector(this.editor.getDOMHelper()).forEach(function(table) {
				var rect = normalizeRect(table.table.getBoundingClientRect());
				if (rect && _this.tableRectMap) _this.tableRectMap.push(__assign(__assign({}, table), { rect }));
			});
		}
	};
	return TableEditPlugin$1;
}();
function defaultTableSelector(domHelper) {
	return domHelper.queryElements("table").filter(function(table) {
		return table.isContentEditable;
	}).map(function(table) {
		return {
			table,
			logicalRoot: null
		};
	});
}
function addParser(domToModelOption, entry, additionalFormatParsers) {
	var _a$5;
	if (!domToModelOption.additionalFormatParsers) domToModelOption.additionalFormatParsers = {};
	if (!domToModelOption.additionalFormatParsers[entry]) domToModelOption.additionalFormatParsers[entry] = [];
	(_a$5 = domToModelOption.additionalFormatParsers[entry]) === null || _a$5 === void 0 || _a$5.push(additionalFormatParsers);
}
function chainSanitizerCallback(map, name, newCallback) {
	var finalCb = typeof newCallback == "function" ? newCallback : function(value) {
		return newCallback ? value : null;
	};
	if (!map[name]) map[name] = finalCb;
	else {
		var originalCallback_1 = map[name];
		map[name] = function(value, tagName) {
			var og = typeof originalCallback_1 == "function" ? originalCallback_1(value, tagName) : originalCallback_1 ? value : false;
			if (!og) return null;
			else return finalCb(og, tagName);
		};
	}
}
var DefaultSanitizers = {
	width: divParagraphSanitizer,
	height: divParagraphSanitizer,
	"inline-size": divParagraphSanitizer,
	"block-size": divParagraphSanitizer
};
function divParagraphSanitizer(value, tagName) {
	var tag = tagName.toLowerCase();
	if (tag == "div" || tag == "p") return null;
	return value;
}
var deprecatedBorderColorParser = function(format) {
	BorderKeys.forEach(function(key) {
		var value = format[key];
		var color = "";
		if (value && DeprecatedColors.some(function(dColor) {
			return value.indexOf(dColor) > -1 && (color = dColor);
		})) format[key] = value.replace(color, "").trimRight();
	});
};
var WORD_ONLINE_TABLE_TEMP_ELEMENT_CLASSES = [
	"TableInsertRowGapBlank",
	"TableColumnResizeHandle",
	"TableCellTopBorderHandle",
	"TableCellLeftBorderHandle",
	"TableHoverColumnHandle",
	"TableHoverRowHandle"
];
var BULLET_LIST_STYLE = "BulletListStyle";
var NUMBER_LIST_STYLE = "NumberListStyle";
var IMAGE_BORDER = "WACImageBorder";
var IMAGE_CONTAINER = "WACImageContainer";
var OUTLINE_ELEMENT = "OutlineElement";
var COMMENT_HIGHLIGHT_CLASS = "CommentHighlightRest";
var COMMENT_HIGHLIGHT_CLICKED_CLASS = "CommentHighlightClicked";
var TEMP_ELEMENTS_CLASSES = __spreadArray(__spreadArray([], __read(WORD_ONLINE_TABLE_TEMP_ELEMENT_CLASSES), false), ["ListMarkerWrappingSpan"], false);
var REMOVE_MARGIN_ELEMENTS = "span." + IMAGE_CONTAINER + ",span." + IMAGE_BORDER + ",." + COMMENT_HIGHLIGHT_CLASS + ",." + COMMENT_HIGHLIGHT_CLICKED_CLASS + "," + WORD_ONLINE_TABLE_TEMP_ELEMENT_CLASSES.map(function(c$1) {
	return "table div[class^=\"" + c$1 + "\"]";
}).join(",");
var WAC_IDENTIFY_SELECTOR = "ul[class^=\"" + BULLET_LIST_STYLE + "\"]>." + OUTLINE_ELEMENT + ",ol[class^=\"" + NUMBER_LIST_STYLE + "\"]>." + OUTLINE_ELEMENT + "," + REMOVE_MARGIN_ELEMENTS;
var documentContainWacElements = function(props) {
	return !!props.fragment.querySelector(WAC_IDENTIFY_SELECTOR);
};
var PastePropertyNames = {
	GOOGLE_SHEET_NODE_NAME: "google-sheets-html-origin",
	PROG_ID_NAME: "ProgId",
	EXCEL_DESKTOP_ATTRIBUTE_NAME: "xmlns:x"
};
var EXCEL_ATTRIBUTE_VALUE = "urn:schemas-microsoft-com:office:excel";
var isExcelDesktopDocument = function(props) {
	return props.htmlAttributes[PastePropertyNames.EXCEL_DESKTOP_ATTRIBUTE_NAME] == EXCEL_ATTRIBUTE_VALUE;
};
var ShadowWorkbookClipboardType = "web data/shadow-workbook";
var isExcelNotNativeEvent = function(props) {
	var _a$5;
	var clipboardData = props.clipboardData;
	return clipboardData.types.includes(ShadowWorkbookClipboardType) && ((_a$5 = clipboardData.htmlFirstLevelChildTags) === null || _a$5 === void 0 ? void 0 : _a$5.length) == 1 && clipboardData.htmlFirstLevelChildTags[0] == "TABLE";
};
var EXCEL_ONLINE_ATTRIBUTE_VALUE = "Excel.Sheet";
var isExcelOnlineDocument = function(props) {
	var htmlAttributes = props.htmlAttributes;
	return htmlAttributes[PastePropertyNames.PROG_ID_NAME] == EXCEL_ONLINE_ATTRIBUTE_VALUE && htmlAttributes[PastePropertyNames.EXCEL_DESKTOP_ATTRIBUTE_NAME] == void 0;
};
var isGoogleSheetDocument = function(props) {
	return !!props.fragment.querySelector(PastePropertyNames.GOOGLE_SHEET_NODE_NAME);
};
var ONE_NOTE_ATTRIBUTE_VALUE = "OneNote.File";
var isOneNoteDesktopDocument = function(props) {
	return props.htmlAttributes[PastePropertyNames.PROG_ID_NAME] == ONE_NOTE_ATTRIBUTE_VALUE;
};
var POWERPOINT_ATTRIBUTE_VALUE = "PowerPoint.Slide";
var isPowerPointDesktopDocument = function(props) {
	return props.htmlAttributes[PastePropertyNames.PROG_ID_NAME] == POWERPOINT_ATTRIBUTE_VALUE;
};
var WORD_ATTRIBUTE_NAME = "xmlns:w";
var WORD_ATTRIBUTE_VALUE = "urn:schemas-microsoft-com:office:word";
var WORD_PROG_ID = "Word.Document";
var isWordDesktopDocument = function(props) {
	var _a$5;
	var htmlAttributes = props.htmlAttributes, clipboardData = props.clipboardData, environment = props.environment;
	return htmlAttributes[WORD_ATTRIBUTE_NAME] == WORD_ATTRIBUTE_VALUE || htmlAttributes[PastePropertyNames.PROG_ID_NAME] == WORD_PROG_ID || !!(environment.isSafari && clipboardData.rawHtml && ((_a$5 = clipboardData.rawHtml) === null || _a$5 === void 0 ? void 0 : _a$5.replace(/ /g, "").indexOf(WORD_ATTRIBUTE_NAME + "=\"" + WORD_ATTRIBUTE_VALUE)) > -1);
};
var shouldConvertToSingleImage = function(props) {
	var _a$5;
	var shouldConvertSingleImage = props.shouldConvertSingleImage, clipboardData = props.clipboardData;
	return shouldConvertSingleImage && ((_a$5 = clipboardData.htmlFirstLevelChildTags) === null || _a$5 === void 0 ? void 0 : _a$5.length) == 1 && clipboardData.htmlFirstLevelChildTags[0] == "IMG";
};
var getSourceFunctions = new Map([
	["wordDesktop", isWordDesktopDocument],
	["excelDesktop", isExcelDesktopDocument],
	["excelOnline", isExcelOnlineDocument],
	["powerPointDesktop", isPowerPointDesktopDocument],
	["wacComponents", documentContainWacElements],
	["googleSheets", isGoogleSheetDocument],
	["singleImage", shouldConvertToSingleImage],
	["excelNonNativeEvent", isExcelNotNativeEvent],
	["oneNoteDesktop", isOneNoteDesktopDocument]
]);
function getPasteSource(event$1, shouldConvertSingleImage, environment) {
	var htmlAttributes = event$1.htmlAttributes, clipboardData = event$1.clipboardData, fragment = event$1.fragment;
	var result = null;
	var param = {
		htmlAttributes,
		fragment,
		shouldConvertSingleImage,
		clipboardData,
		environment
	};
	getSourceFunctions.forEach(function(func, key) {
		if (!result && func(param)) result = key;
	});
	return result !== null && result !== void 0 ? result : "default";
}
var SUPPORTED_PROTOCOLS = [
	"http:",
	"https:",
	"notes:",
	"mailto:",
	"onenote:"
];
var INVALID_LINKS_REGEX = /^file:\/\/\/[a-zA-Z\/]/i;
var parseLink = function(format, element) {
	if (!isElementOfType(element, "a")) return;
	var url;
	try {
		url = new URL(element.href);
	} catch (_a$5) {
		url = void 0;
	}
	if (url && SUPPORTED_PROTOCOLS.indexOf(url.protocol) === -1 || INVALID_LINKS_REGEX.test(element.href)) {
		element.removeAttribute("href");
		format.href = "";
	}
};
var pasteButtonProcessor = function(group, element, context) {
	var format = {};
	parseFormat(element, context.formatParsers.segment, format, context);
	processTextNodesRecursively(group, element, context, format);
};
function processTextNodesRecursively(group, node, context, format) {
	if (node.nodeType === Node.TEXT_NODE) addSegment(group, createText(node.nodeValue || "", format));
	else if (isNodeOfType(node, "ELEMENT_NODE")) {
		var newFormat = __assign({}, format);
		parseFormat(node, context.formatParsers.segment, newFormat, context);
		for (var i$1 = 0; i$1 < node.childNodes.length; i$1++) processTextNodesRecursively(group, node.childNodes[i$1], context, newFormat);
	}
}
function setProcessor(domToModelOption, entry, processorOverride) {
	if (!domToModelOption.processorOverride) domToModelOption.processorOverride = {};
	domToModelOption.processorOverride[entry] = processorOverride;
}
var LAST_TD_END_REGEX = /<\/\s*td\s*>((?!<\/\s*tr\s*>)[\s\S])*$/i;
var LAST_TR_END_REGEX = /<\/\s*tr\s*>((?!<\/\s*table\s*>)[\s\S])*$/i;
var LAST_TR_REGEX = /<tr[^>]*>[^<]*/i;
var LAST_TABLE_REGEX = /<table[^>]*>[^<]*/i;
var TABLE_SELECTOR = "table";
var DEFAULT_BORDER_STYLE = "solid 1px #d4d4d4";
function processPastedContentFromExcel(event$1, domCreator, allowExcelNoBorderTable, isNativeEvent) {
	var fragment = event$1.fragment, htmlBefore = event$1.htmlBefore, htmlAfter = event$1.htmlAfter, clipboardData = event$1.clipboardData;
	if (isNativeEvent) validateExcelFragment(fragment, domCreator, htmlBefore, clipboardData, htmlAfter);
	var firstChild = fragment.firstChild;
	if (isNodeOfType(firstChild, "ELEMENT_NODE") && firstChild.tagName == "div" && firstChild.firstChild) {
		if (Array.from(firstChild.childNodes).every(function(child$1) {
			var tagName = isNodeOfType(child$1, "ELEMENT_NODE") && child$1.tagName;
			return tagName == "META" ? true : tagName == "TABLE" ? child$1 == firstChild.lastChild : false;
		}) && firstChild.lastChild) event$1.fragment.replaceChildren(firstChild.lastChild);
	}
	setupExcelTableHandlers(event$1, allowExcelNoBorderTable, isNativeEvent);
}
function validateExcelFragment(fragment, domCreator, htmlBefore, clipboardData, htmlAfter) {
	var result = !fragment.querySelector(TABLE_SELECTOR) && domCreator.htmlToDOM(htmlBefore + clipboardData.html + htmlAfter);
	if (result && result.querySelector(TABLE_SELECTOR)) moveChildNodes(fragment, result === null || result === void 0 ? void 0 : result.body);
	else {
		var html$1 = clipboardData.html ? excelHandler(clipboardData.html, htmlBefore) : void 0;
		if (html$1 && clipboardData.html != html$1) {
			var doc = domCreator.htmlToDOM(html$1);
			moveChildNodes(fragment, doc === null || doc === void 0 ? void 0 : doc.body);
		}
	}
}
function excelHandler(html$1, htmlBefore) {
	try {
		if (html$1.match(LAST_TD_END_REGEX)) {
			var trMatch = htmlBefore.match(LAST_TR_REGEX);
			html$1 = (trMatch ? trMatch[0] : "<TR>") + html$1 + "</TR>";
		}
		if (html$1.match(LAST_TR_END_REGEX)) {
			var tableMatch = htmlBefore.match(LAST_TABLE_REGEX);
			html$1 = (tableMatch ? tableMatch[0] : "<TABLE>") + html$1 + "</TABLE>";
		}
	} finally {
		return html$1;
	}
}
function setupExcelTableHandlers(event$1, allowExcelNoBorderTable, isNativeEvent) {
	addParser(event$1.domToModelOption, "tableCell", function(format, element) {
		if (!allowExcelNoBorderTable && (element.style.borderStyle === "none" || !isNativeEvent && element.style.borderStyle == "")) {
			format.borderBottom = DEFAULT_BORDER_STYLE;
			format.borderLeft = DEFAULT_BORDER_STYLE;
			format.borderRight = DEFAULT_BORDER_STYLE;
			format.borderTop = DEFAULT_BORDER_STYLE;
		}
	});
	setProcessor(event$1.domToModelOption, "child", childProcessor);
}
var childProcessor = function(group, element, context) {
	var segmentFormat = __assign({}, context.segmentFormat);
	if (group.blockGroupType === "TableCell" && group.format.textColor && !context.segmentFormat.textColor) context.segmentFormat.textColor = group.format.textColor;
	context.defaultElementProcessors.child(group, element, context);
	if (group.blockGroupType === "TableCell" && group.format.textColor) {
		context.segmentFormat = segmentFormat;
		delete group.format.textColor;
	}
};
var OrderedListStyleMap$1 = {
	1: "decimal",
	a: "lower-alpha",
	A: "upper-alpha",
	i: "lower-roman",
	I: "upper-roman"
};
var UnorderedListStyleMap = {
	disc: "disc",
	circle: "circle",
	square: "square"
};
function processPastedContentFromOneNote(event$1) {
	setProcessor(event$1.domToModelOption, "ol", processOrderedList);
	setProcessor(event$1.domToModelOption, "ul", processUnorderedList);
	setProcessor(event$1.domToModelOption, "li", processListItem);
}
var processOrderedList = function(group, element, cmContext) {
	var _a$5, _b$1;
	var context = ensureOneNoteListContext(cmContext);
	if (context.oneNoteListContext) {
		var typeOfList = element.getAttribute("type");
		if (typeOfList) {
			var listStyle = OrderedListStyleMap$1[typeOfList];
			var startNumberOverride = parseInt(element.getAttribute("start") || "1") || 1;
			context.oneNoteListContext.listStyleType = listStyle;
			context.oneNoteListContext.startNumberOverride = startNumberOverride;
		}
	}
	(_b$1 = (_a$5 = context.defaultElementProcessors).ol) === null || _b$1 === void 0 || _b$1.call(_a$5, group, element, context);
};
var processUnorderedList = function(group, element, cmContext) {
	var _a$5, _b$1;
	var context = ensureOneNoteListContext(cmContext);
	if (context.oneNoteListContext) {
		var typeOfList = element.getAttribute("type");
		if (typeOfList) {
			var listStyle = UnorderedListStyleMap[typeOfList];
			context.oneNoteListContext.listStyleType = listStyle;
		}
	}
	(_b$1 = (_a$5 = context.defaultElementProcessors).ul) === null || _b$1 === void 0 || _b$1.call(_a$5, group, element, context);
};
var processListItem = function(group, element, cmContext) {
	var _a$5, _b$1;
	var context = ensureOneNoteListContext(cmContext);
	var removeStartNumberOverride = false;
	if (context.oneNoteListContext) {
		var _c$1 = context.oneNoteListContext, listStyleType = _c$1.listStyleType, startNumberOverride = _c$1.startNumberOverride;
		if (listStyleType) {
			var lastLevel = context.listFormat.levels[context.listFormat.levels.length - 1];
			lastLevel.format.listStyleType = listStyleType;
			if (startNumberOverride) {
				removeStartNumberOverride = true;
				lastLevel.format.startNumberOverride = startNumberOverride;
				delete context.oneNoteListContext.startNumberOverride;
			}
			delete context.oneNoteListContext.listStyleType;
		}
	}
	(_b$1 = (_a$5 = context.defaultElementProcessors).li) === null || _b$1 === void 0 || _b$1.call(_a$5, group, element, context);
	if (removeStartNumberOverride) delete context.listFormat.levels[context.listFormat.levels.length - 1].format.startNumberOverride;
};
function ensureOneNoteListContext(cmContext) {
	var context = cmContext;
	if (!context.oneNoteListContext) context.oneNoteListContext = {};
	return context;
}
var removeNegativeTextIndentParser = function(format) {
	var _a$5;
	if ((_a$5 = format.textIndent) === null || _a$5 === void 0 ? void 0 : _a$5.startsWith("-")) delete format.textIndent;
};
var removeMargin = function(format) {
	delete format.marginLeft;
};
function setupListFormat(listType, element, context, listDepth, listFormat, group, additionalParsers) {
	if (additionalParsers === void 0) additionalParsers = [];
	var newLevel = createListLevel(listType);
	parseFormat(element, context.formatParsers.listLevel, newLevel.format, context);
	parseFormat(element, additionalParsers.concat(removeMargin), newLevel.format, context);
	if (listDepth > listFormat.levels.length) while (listDepth != listFormat.levels.length) listFormat.levels.push(newLevel);
	else {
		listFormat.levels.splice(listDepth, listFormat.levels.length - 1);
		listFormat.levels[listDepth - 1] = newLevel;
	}
	listFormat.listParent = group;
}
function processAsListItem(context, element, group, listFormatMetadata, bulletElement, beforeProcessingChildren) {
	var listFormat = context.listFormat;
	var lastLevel = listFormat.levels[listFormat.levels.length - 1];
	if (listFormatMetadata && lastLevel) updateListMetadata(lastLevel, function(metadata) {
		return Object.assign({}, metadata, listFormatMetadata);
	});
	var listItem = createListItem(listFormat.levels, context.segmentFormat);
	parseFormat(element, context.formatParsers.segmentOnBlock, context.segmentFormat, context);
	parseFormat(element, context.formatParsers.listItemElement, listItem.format, context);
	parseFormat(element, [removeNegativeTextIndentParser, nonListElementParser], listItem.format, context);
	if (bulletElement) {
		var format = __assign({}, context.segmentFormat);
		parseFormat(bulletElement, context.formatParsers.segmentOnBlock, format, context);
		listItem.formatHolder.format = format;
	}
	beforeProcessingChildren === null || beforeProcessingChildren === void 0 || beforeProcessingChildren(listItem);
	context.elementProcessors.child(listItem, element, context);
	addBlock(group, listItem);
}
var nonListElementParser = function(format, element, _context, defaultStyle) {
	if (!isElementOfType(element, "li")) Object.keys(defaultStyle).forEach(function(keyInput) {
		var key = keyInput;
		var formatKey = keyInput;
		if (key != "display" && format[formatKey] != void 0 && format[formatKey] == defaultStyle[key]) delete format[formatKey];
	});
};
var BulletSelector = "* > span > span[style*=mso-special-format]";
var MsOfficeSpecialFormat = "mso-special-format";
var CssStyleKey = "style";
var MsoSpecialFormatRegex = /mso-special-format:\s*([^;]*)/;
var clearListItemStyles = function(format) {
	delete format.textAlign;
	delete format.marginLeft;
	delete format.paddingLeft;
};
function processPastedContentFromPowerPoint(event$1, domCreator) {
	var fragment = event$1.fragment, clipboardData = event$1.clipboardData, domToModelOption = event$1.domToModelOption;
	if (clipboardData.html && !clipboardData.text && clipboardData.image) {
		var doc = domCreator.htmlToDOM(clipboardData.html);
		moveChildNodes(fragment, doc === null || doc === void 0 ? void 0 : doc.body);
	}
	addParser(domToModelOption, "block", removeNegativeTextIndentParser);
	setProcessor(domToModelOption, "element", function(group, element, context) {
		var _a$5, _b$1;
		if ((element.getAttribute(CssStyleKey) || "").includes(MsOfficeSpecialFormat) && context.listFormat.levels.length > 0) return;
		var bulletElement = element.querySelector(BulletSelector);
		if (bulletElement) {
			var _c$1 = extractPowerPointListInfo(element, bulletElement), depth = _c$1.depth, unorderedBulletType = _c$1.unorderedBulletType, orderedBulletType = _c$1.orderedBulletType, startNumberOverrideOrBullet_1 = _c$1.startNumberOverrideOrBullet, isOrderedList = _c$1.isOrderedList, isNewList_1 = _c$1.isNewList;
			setupListFormat(isOrderedList ? "OL" : "UL", element, context, depth, context.listFormat, group, [clearListItemStyles]);
			processAsListItem(context, element, group, {
				unorderedStyleType: !isOrderedList && unorderedBulletType ? BulletListType[unorderedBulletType] : void 0,
				orderedStyleType: isOrderedList && orderedBulletType ? NumberingListType[orderedBulletType] : void 0
			}, bulletElement, function(listItem) {
				if (isNewList_1) listItem.levels[listItem.levels.length - 1].format.startNumberOverride = parseInt(startNumberOverrideOrBullet_1);
				clearListItemStyles(listItem.levels[listItem.levels.length - 1].format);
				clearListItemStyles(listItem.format);
			});
		} else (_b$1 = (_a$5 = context.defaultElementProcessors).element) === null || _b$1 === void 0 || _b$1.call(_a$5, group, element, context);
	});
}
function extractPowerPointListInfo(element, bulletElement) {
	var className = element.className.substring(1) || "0";
	var depth = parseInt(className) + 1;
	var msoSpecialFormat = (bulletElement.getAttribute(CssStyleKey) || "").match(MsoSpecialFormatRegex);
	var _a$5 = __read((msoSpecialFormat === null || msoSpecialFormat === void 0 ? void 0 : msoSpecialFormat[1].replace("\"", "").split("\\,")) || [], 2), bulletTypeHtml = _a$5[0], startNumberOverrideOrBullet = _a$5[1];
	var isOrderedList = OrderedListStyleMap.has(bulletTypeHtml);
	var unorderedBulletType = UnorderedBullets.get(bulletElement.innerText);
	var orderedBulletType = OrderedListStyleMap.get(bulletTypeHtml);
	return {
		depth,
		unorderedBulletType,
		orderedBulletType,
		startNumberOverrideOrBullet,
		isOrderedList,
		isNewList: isOrderedList && !!orderedBulletType && bulletElement.innerText === getPptListStart(orderedBulletType, startNumberOverrideOrBullet)
	};
}
var UnorderedBullets = new Map([
	["•", "Disc"],
	["o", "Circle"],
	["§", "Square"],
	["q", "BoxShadow"],
	["v", "Xrhombus"],
	["Ø", "ShortArrow"],
	["ü", "CheckMark"]
]);
var OrderedListStyleMap = new Map([
	["numbullet1", "UpperAlpha"],
	["numbullet2", "DecimalParenthesis"],
	["numbullet3", "Decimal"],
	["numbullet7", "UpperRoman"],
	["numbullet9", "LowerAlphaParenthesis"],
	["numbullet0", "LowerAlpha"],
	["numbullet6", "LowerRoman"]
]);
function getPptListStart(orderedBulletType, startNumberOverride) {
	var bullet = getOrderedListNumberStr(NumberingListType[orderedBulletType], parseInt(startNumberOverride));
	switch (orderedBulletType) {
		case "Decimal":
		case "UpperAlpha":
		case "LowerAlpha":
		case "UpperRoman":
		case "LowerRoman": return bullet + ".";
		case "DecimalParenthesis":
		case "LowerAlphaParenthesis": return bullet + ")";
		default: return;
	}
}
var FORMATING_REGEX = /[\n\t'{}"]+/g;
var STYLE_TAG = "<style";
var STYLE_TAG_END = "</style>";
var nonWordCharacterRegex = /\W/;
function extractStyleTagsFromHtml(htmlContent) {
	var _a$5;
	var styles = [];
	var _b$1 = extractHtmlIndexes(htmlContent), styleIndex = _b$1.styleIndex, styleEndIndex = _b$1.styleEndIndex;
	while (styleIndex >= 0 && styleEndIndex >= 0) {
		var styleContent = htmlContent.substring(styleIndex + STYLE_TAG.length, styleEndIndex).trim();
		styles.push(styleContent);
		_a$5 = extractHtmlIndexes(htmlContent, styleEndIndex + 1), styleIndex = _a$5.styleIndex, styleEndIndex = _a$5.styleEndIndex;
	}
	return styles;
}
function extractHtmlIndexes(html$1, startIndex) {
	if (startIndex === void 0) startIndex = 0;
	var htmlLowercase = html$1.toLowerCase();
	var styleIndex = htmlLowercase.indexOf(STYLE_TAG, startIndex);
	var currentIndex = styleIndex + STYLE_TAG.length;
	var nextChar = html$1.substring(currentIndex, currentIndex + 1);
	while (!nonWordCharacterRegex.test(nextChar) && styleIndex > -1) {
		styleIndex = htmlLowercase.indexOf(STYLE_TAG, styleIndex + 1);
		currentIndex = styleIndex + STYLE_TAG.length;
		nextChar = html$1.substring(currentIndex, currentIndex + 1);
	}
	var styleEndIndex = htmlLowercase.indexOf(STYLE_TAG_END, startIndex);
	return {
		styleIndex,
		styleEndIndex
	};
}
function getStyleMetadata(ev) {
	var metadataMap = /* @__PURE__ */ new Map();
	extractStyleTagsFromHtml(ev.htmlBefore || ev.clipboardData.rawHtml || "").forEach(function(text) {
		var index$1 = 0;
		var _loop_1 = function() {
			var indexAt = text.indexOf("@", index$1 + 1);
			var indexCurlyEnd = text.indexOf("}", indexAt);
			var indexCurlyStart = text.indexOf("{", indexAt);
			index$1 = indexAt;
			var metadataName = text.substring(indexAt + 1, indexCurlyStart).replace(FORMATING_REGEX, "").replace("list", "").trimRight().trimLeft();
			var dataName = text.substring(indexCurlyStart, indexCurlyEnd + 1).trimLeft().trimRight();
			var record = {};
			dataName.split(";").forEach(function(entry) {
				var _a$5 = __read(entry.split(":"), 2), key = _a$5[0], value = _a$5[1];
				if (key && value) {
					var formatedKey = key.replace(FORMATING_REGEX, "").trimRight().trimLeft();
					record[formatedKey] = value.replace(FORMATING_REGEX, "").trimRight().trimLeft();
				}
			});
			var data = {
				"mso-level-number-format": record["mso-level-number-format"],
				"mso-level-start-at": record["mso-level-start-at"] || "1",
				"mso-level-text": record["mso-level-text"]
			};
			if (getObjectKeys(data).some(function(key) {
				return !!data[key];
			})) metadataMap.set(metadataName, data);
		};
		while (index$1 >= 0) _loop_1();
	});
	return metadataMap;
}
function getStyles(element) {
	var result = {};
	((element === null || element === void 0 ? void 0 : element.getAttribute("style")) || "").split(";").forEach(function(pair) {
		var valueIndex = pair.indexOf(":");
		var name = pair.slice(0, valueIndex);
		var value = pair.slice(valueIndex + 1);
		if (name && value) result[name.trim()] = value.trim();
	});
	return result;
}
var MSO_COMMENT_ANCHOR_HREF_REGEX = /#_msocom_/;
var MSO_SPECIAL_CHARACTER = "mso-special-character";
var MSO_SPECIAL_CHARACTER_COMMENT = "comment";
var MSO_ELEMENT = "mso-element";
var MSO_ELEMENT_COMMENT_LIST = "comment-list";
function processWordComments(styles, element) {
	return styles[MSO_SPECIAL_CHARACTER] == MSO_SPECIAL_CHARACTER_COMMENT || isElementOfType(element, "a") && MSO_COMMENT_ANCHOR_HREF_REGEX.test(element.href) || styles[MSO_ELEMENT] == MSO_ELEMENT_COMMENT_LIST;
}
var MSO_LIST = "mso-list";
var MSO_LIST_IGNORE = "ignore";
var WORD_FIRST_LIST = "l0";
var TEMPLATE_VALUE_REGEX = /%[0-9a-zA-Z]+/g;
var BULLET_METADATA = "bullet";
function processWordList(styles, group, element, context, metadata) {
	var _a$5;
	var listFormat = context.listFormat;
	if (!listFormat.wordKnownLevels) listFormat.wordKnownLevels = /* @__PURE__ */ new Map();
	var wordListStyle = styles[MSO_LIST] || "";
	if (wordListStyle.toLowerCase() === MSO_LIST_IGNORE) return true;
	var _b$1 = __read(wordListStyle.split(" "), 2), lNumber = _b$1[0], level = _b$1[1];
	listFormat.wordLevel = level && parseInt(level.substr(5));
	listFormat.wordList = lNumber || WORD_FIRST_LIST;
	if (listFormat.levels.length == 0) listFormat.levels = listFormat.wordList && listFormat.wordKnownLevels.get(listFormat.wordList) || [];
	if (wordListStyle && group && typeof listFormat.wordLevel === "number") {
		var wordLevel = listFormat.wordLevel, wordList = listFormat.wordList;
		var listMetadata_1 = metadata.get(lNumber + ":" + level);
		var listType_1 = ((_a$5 = listMetadata_1 === null || listMetadata_1 === void 0 ? void 0 : listMetadata_1["mso-level-number-format"]) === null || _a$5 === void 0 ? void 0 : _a$5.toLowerCase()) != BULLET_METADATA ? "OL" : "UL";
		setupListFormat(listType_1, element, context, wordLevel, listFormat, group, [wordListPaddingParser]);
		listFormat.levels[listFormat.levels.length - 1].format.wordList = wordList;
		var bullet = getBulletFromMetadata(listMetadata_1, listType_1);
		processAsListItem(context, element, group, bullet ? {
			unorderedStyleType: listType_1 == "UL" ? bullet : void 0,
			orderedStyleType: listType_1 == "OL" ? bullet : void 0
		} : void 0, getBulletElement(element), function(listItem) {
			if (listType_1 == "OL") setStartNumber(listItem, context, listMetadata_1, element);
		});
		if (listFormat.levels.length > 0 && listFormat.wordKnownLevels.get(wordList) != listFormat.levels) listFormat.wordKnownLevels.set(wordList, __spreadArray([], __read(listFormat.levels), false));
		return true;
	}
	return false;
}
function getBulletFromMetadata(listMetadata, listType) {
	var templateType = (listMetadata === null || listMetadata === void 0 ? void 0 : listMetadata["mso-level-number-format"]) || "decimal";
	var templateFinal;
	if (listMetadata === null || listMetadata === void 0 ? void 0 : listMetadata["mso-level-text"]) {
		var templateValue = "";
		switch (templateType) {
			case "alpha-upper":
				templateValue = "UpperAlpha";
				break;
			case "alpha-lower":
				templateValue = "LowerAlpha";
				break;
			case "roman-lower":
				templateValue = "LowerRoman";
				break;
			case "roman-upper":
				templateValue = "UpperRoman";
				break;
			default:
				templateValue = "Number";
				break;
		}
		templateFinal = "\"" + (listMetadata["mso-level-text"] || "").replace("\\", "").replace("\"", "").replace(TEMPLATE_VALUE_REGEX, "${" + templateValue + "}") + " \"";
	} else switch (templateType) {
		case "alpha-lower":
			templateFinal = "lower-alpha";
			break;
		case "roman-lower":
			templateFinal = "lower-roman";
			break;
		case "roman-upper":
			templateFinal = "upper-roman";
			break;
		default:
			templateFinal = "decimal";
			break;
	}
	return getListStyleTypeFromString(listType, templateFinal);
}
function setStartNumber(listItem, context, listMetadata, element) {
	var _a$5, _b$1;
	var _c$1 = context.listFormat, listParent = _c$1.listParent, wordList = _c$1.wordList, wordKnownLevels = _c$1.wordKnownLevels, wordLevel = _c$1.wordLevel, levels = _c$1.levels;
	var block = getLastNotEmptyBlock(listParent);
	if (((block === null || block === void 0 ? void 0 : block.blockType) != "BlockGroup" || block.blockGroupType != "ListItem" || wordLevel && ((_b$1 = (_a$5 = block.levels[wordLevel]) === null || _a$5 === void 0 ? void 0 : _a$5.format) === null || _b$1 === void 0 ? void 0 : _b$1.wordList) != wordList) && wordList) {
		var start = (listMetadata === null || listMetadata === void 0 ? void 0 : listMetadata["mso-level-start-at"]) ? parseInt(listMetadata["mso-level-start-at"]) : NaN;
		var knownLevel = (wordKnownLevels === null || wordKnownLevels === void 0 ? void 0 : wordKnownLevels.get(wordList)) || [];
		if (start != void 0 && !isNaN(start) && knownLevel.length != levels.length) listItem.levels[listItem.levels.length - 1].format.startNumberOverride = start;
		else if (isElementOfType(element, "li") && isNodeOfType(element.parentElement, "ELEMENT_NODE") && isElementOfType(element.parentElement, "ol") && element.parentElement.firstElementChild == element && knownLevel.length != element.parentElement.start) listItem.levels[listItem.levels.length - 1].format.startNumberOverride = element.parentElement.start;
	}
}
function getLastNotEmptyBlock(listParent) {
	for (var index$1 = ((listParent === null || listParent === void 0 ? void 0 : listParent.blocks.length) || 0) - 1; index$1 > 0; index$1--) {
		var result = listParent === null || listParent === void 0 ? void 0 : listParent.blocks[index$1];
		if (result && !isEmpty(result)) return result;
	}
}
function wordListPaddingParser(format, element) {
	if (element.style.marginLeft && parseInt(element.style.marginLeft) != 0) format.paddingLeft = "0px";
	if (element.style.marginRight && parseInt(element.style.marginRight) != 0) format.paddingRight = "0px";
}
function getBulletElement(element) {
	var firstChild = element.firstElementChild;
	var isBulletElement = false;
	if (firstChild) for (var i$1 = 0; i$1 < firstChild.childNodes.length; i$1++) {
		var child$1 = firstChild.childNodes[i$1];
		if (isNodeOfType(child$1, "ELEMENT_NODE")) {
			if ((getStyles(child$1)[MSO_LIST] || "").toLowerCase() === MSO_LIST_IGNORE) {
				isBulletElement = true;
				break;
			}
		}
	}
	return firstChild && isBulletElement ? firstChild : void 0;
}
var PERCENTAGE_REGEX = /%/;
var DEFAULT_BROWSER_LINE_HEIGHT_PERCENTAGE = 1.2;
function processPastedContentFromWordDesktop(ev) {
	var metadataMap = getStyleMetadata(ev);
	setProcessor(ev.domToModelOption, "element", wordDesktopElementProcessor(metadataMap));
	addParser(ev.domToModelOption, "block", adjustPercentileLineHeight);
	addParser(ev.domToModelOption, "block", removeNegativeTextIndentParser);
	addParser(ev.domToModelOption, "listLevel", listLevelParser);
	addParser(ev.domToModelOption, "container", wordTableParser);
	addParser(ev.domToModelOption, "table", wordTableParser);
}
var wordDesktopElementProcessor = function(metadataKey) {
	return function(group, element, context) {
		var styles = getStyles(element);
		if (!(processWordList(styles, group, element, context, metadataKey) || processWordComments(styles, element))) context.defaultElementProcessors.element(group, element, context);
	};
};
function adjustPercentileLineHeight(format, element) {
	var parsedLineHeight;
	if (PERCENTAGE_REGEX.test(element.style.lineHeight) && !isNaN(parsedLineHeight = parseInt(element.style.lineHeight))) format.lineHeight = (DEFAULT_BROWSER_LINE_HEIGHT_PERCENTAGE * (parsedLineHeight / 100)).toString();
}
var listLevelParser = function(format, element, _context, defaultStyle) {
	if (element.style.marginLeft != "") format.marginLeft = defaultStyle.marginLeft;
	format.marginBottom = void 0;
};
var wordTableParser = function(format, element) {
	var _a$5;
	if ((_a$5 = format.marginLeft) === null || _a$5 === void 0 ? void 0 : _a$5.startsWith("-")) delete format.marginLeft;
	if (format.htmlAlign) delete format.htmlAlign;
};
var LIST_ELEMENT_TAGS = [
	"UL",
	"OL",
	"LI"
];
var LIST_ELEMENT_SELECTOR = LIST_ELEMENT_TAGS.join(",");
var wacSubSuperParser = function(format, element) {
	var verticalAlign = element.style.verticalAlign;
	if (verticalAlign === "super") format.superOrSubScriptSequence = "super";
	if (verticalAlign === "sub") format.superOrSubScriptSequence = "sub";
};
var wacElementProcessor = function(group, element, context) {
	var elementTag = element.tagName;
	if (element.matches(REMOVE_MARGIN_ELEMENTS)) {
		element.style.removeProperty("display");
		element.style.removeProperty("margin");
	}
	if (element.classList.contains("ListContainerWrapper")) {
		context.elementProcessors.child(group, element, context);
		return;
	}
	if (TEMP_ELEMENTS_CLASSES.some(function(className) {
		return element.classList.contains(className);
	})) return;
	else if (shouldClearListContext(elementTag, element, context)) {
		var listFormat = context.listFormat;
		listFormat.levels = [];
		listFormat.listParent = void 0;
	}
	context.defaultElementProcessors.element(group, element, context);
};
var wacLiElementProcessor = function(group, element, context) {
	var _a$5, _b$1, _c$1, _d$1, _e$1;
	var level = parseInt((_a$5 = element.getAttribute("data-aria-level")) !== null && _a$5 !== void 0 ? _a$5 : "");
	var listFormat = context.listFormat;
	var newLevel = createListLevel(((_b$1 = listFormat.levels[context.listFormat.levels.length - 1]) === null || _b$1 === void 0 ? void 0 : _b$1.listType) || ((_c$1 = element.closest("ol,ul")) === null || _c$1 === void 0 ? void 0 : _c$1.tagName.toUpperCase()), context.blockFormat);
	parseFormat(element, context.formatParsers.listLevelThread, newLevel.format, context);
	parseFormat(element, context.formatParsers.listLevel, newLevel.format, context);
	context.listFormat.levels = listFormat.currentListLevels || context.listFormat.levels;
	if (level > 0) if (level > context.listFormat.levels.length) while (level != context.listFormat.levels.length) context.listFormat.levels.push(newLevel);
	else {
		context.listFormat.levels.splice(level, context.listFormat.levels.length - 1);
		context.listFormat.levels[level - 1] = newLevel;
	}
	(_e$1 = (_d$1 = context.defaultElementProcessors).li) === null || _e$1 === void 0 || _e$1.call(_d$1, group, element, context);
	var listParent = listFormat.listParent;
	if (listParent) {
		var lastblock = listParent.blocks[listParent.blocks.length - 1];
		if (lastblock.blockType == "BlockGroup" && lastblock.blockGroupType == "ListItem") {
			var currentLevel = lastblock.levels[lastblock.levels.length - 1];
			updateStartOverride(currentLevel, element, context);
		}
	}
	var newLevels = [];
	listFormat.levels.forEach(function(v$1) {
		var newValue = {
			dataset: __assign({}, v$1.dataset),
			format: __assign({}, v$1.format),
			listType: v$1.listType
		};
		newLevels.push(newValue);
	});
	listFormat.currentListLevels = newLevels;
	listFormat.levels = [];
};
var wacListItemParser = function(format, element) {
	if (element.style.display === "block") format.displayForDummyItem = void 0;
	format.marginLeft = void 0;
	format.marginRight = void 0;
};
var wacListLevelParser = function(format) {
	format.marginLeft = void 0;
	format.paddingLeft = void 0;
};
function shouldClearListContext(elementTag, element, context) {
	return context.listFormat.levels.length > 0 && LIST_ELEMENT_TAGS.every(function(tag) {
		return tag != elementTag;
	}) && !element.closest(LIST_ELEMENT_SELECTOR);
}
var wacCommentParser = function(format, element) {
	if (element.className.includes("CommentHighlightRest") || element.className.includes("CommentHighlightClicked")) delete format.backgroundColor;
};
function processPastedContentWacComponents(ev) {
	addParser(ev.domToModelOption, "segment", wacSubSuperParser);
	addParser(ev.domToModelOption, "listItemThread", wacListItemParser);
	addParser(ev.domToModelOption, "listItemElement", wacListItemParser);
	addParser(ev.domToModelOption, "listLevel", wacListLevelParser);
	addParser(ev.domToModelOption, "container", wacContainerParser);
	addParser(ev.domToModelOption, "table", wacContainerParser);
	addParser(ev.domToModelOption, "segment", wacCommentParser);
	setProcessor(ev.domToModelOption, "element", wacElementProcessor);
	setProcessor(ev.domToModelOption, "li", wacLiElementProcessor);
}
var wacContainerParser = function(format, element) {
	if (element.style.marginLeft.startsWith("-")) delete format.marginLeft;
};
function updateStartOverride(currentLevel, element, ctx) {
	if (!currentLevel || currentLevel.listType == "UL") return;
	var list = element.closest("ol");
	var listFormat = ctx.listFormat;
	var _a$5 = __read(extractWordListMetadata(list, element), 2), start = _a$5[0], listLevel = _a$5[1];
	if (!listFormat.listItemThread) listFormat.listItemThread = [];
	var thread = listFormat.listItemThread[listLevel];
	if (thread && start - thread != 1) currentLevel.format.startNumberOverride = start;
	listFormat.listItemThread[listLevel] = start;
}
function extractWordListMetadata(list, item) {
	var itemIndex = item && Array.from((list === null || list === void 0 ? void 0 : list.querySelectorAll("li")) || []).indexOf(item);
	return [parseInt((list === null || list === void 0 ? void 0 : list.getAttribute("start")) || "1") + (itemIndex && itemIndex > 0 ? itemIndex : 0), parseInt((item === null || item === void 0 ? void 0 : item.getAttribute("data-aria-level")) || "")];
}
var PastePlugin = function() {
	function PastePlugin$1(allowExcelNoBorderTable, domToModelForSanitizing) {
		if (domToModelForSanitizing === void 0) domToModelForSanitizing = {
			styleSanitizers: DefaultSanitizers,
			additionalAllowedTags: [],
			additionalDisallowedTags: [],
			attributeSanitizers: {}
		};
		this.allowExcelNoBorderTable = allowExcelNoBorderTable;
		this.domToModelForSanitizing = domToModelForSanitizing;
		this.editor = null;
	}
	PastePlugin$1.prototype.getName = function() {
		return "Paste";
	};
	PastePlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
	};
	PastePlugin$1.prototype.dispose = function() {
		this.editor = null;
	};
	PastePlugin$1.prototype.onPluginEvent = function(event$1) {
		if (!this.editor || event$1.eventType != "beforePaste") return;
		if (!event$1.domToModelOption) return;
		var pasteSource = getPasteSource(event$1, false, this.editor.getEnvironment());
		var pasteType = event$1.pasteType;
		switch (pasteSource) {
			case "wordDesktop":
				processPastedContentFromWordDesktop(event$1);
				break;
			case "wacComponents":
				processPastedContentWacComponents(event$1);
				break;
			case "excelOnline":
			case "excelDesktop":
			case "excelNonNativeEvent":
				if (pasteType === "normal" || pasteType === "mergeFormat") processPastedContentFromExcel(event$1, this.editor.getDOMCreator(), !!this.allowExcelNoBorderTable, pasteSource != "excelNonNativeEvent");
				break;
			case "googleSheets":
				event$1.domToModelOption.additionalAllowedTags.push(PastePropertyNames.GOOGLE_SHEET_NODE_NAME);
				break;
			case "powerPointDesktop":
				processPastedContentFromPowerPoint(event$1, this.editor.getDOMCreator());
				break;
			case "oneNoteDesktop":
				processPastedContentFromOneNote(event$1);
				break;
		}
		addParser(event$1.domToModelOption, "link", parseLink);
		addParser(event$1.domToModelOption, "tableCell", deprecatedBorderColorParser);
		addParser(event$1.domToModelOption, "tableCell", tableBorderParser);
		addParser(event$1.domToModelOption, "table", deprecatedBorderColorParser);
		setProcessor(event$1.domToModelOption, "button", pasteButtonProcessor);
		if (pasteType === "mergeFormat") {
			addParser(event$1.domToModelOption, "block", blockElementParser);
			addParser(event$1.domToModelOption, "listLevel", blockElementParser);
		}
		this.setEventSanitizers(event$1);
	};
	PastePlugin$1.prototype.setEventSanitizers = function(event$1) {
		var _a$5, _b$1;
		if (this.domToModelForSanitizing) {
			var _c$1 = this.domToModelForSanitizing, styleSanitizers_1 = _c$1.styleSanitizers, attributeSanitizers_1 = _c$1.attributeSanitizers, additionalAllowedTags = _c$1.additionalAllowedTags, additionalDisallowedTags = _c$1.additionalDisallowedTags;
			getObjectKeys(styleSanitizers_1).forEach(function(key) {
				return chainSanitizerCallback(event$1.domToModelOption.styleSanitizers, key, styleSanitizers_1[key]);
			});
			getObjectKeys(attributeSanitizers_1).forEach(function(key) {
				return chainSanitizerCallback(event$1.domToModelOption.attributeSanitizers, key, attributeSanitizers_1[key]);
			});
			(_a$5 = event$1.domToModelOption.additionalAllowedTags).push.apply(_a$5, __spreadArray([], __read(additionalAllowedTags), false));
			(_b$1 = event$1.domToModelOption.additionalDisallowedTags).push.apply(_b$1, __spreadArray([], __read(additionalDisallowedTags), false));
		}
	};
	return PastePlugin$1;
}();
var blockElementParser = function(format, element) {
	if (element.style.backgroundColor) delete format.backgroundColor;
};
var ElementBorderKeys = new Map([
	["borderTop", {
		w: "borderTopWidth",
		s: "borderTopStyle",
		c: "borderTopColor"
	}],
	["borderRight", {
		w: "borderRightWidth",
		s: "borderRightStyle",
		c: "borderRightColor"
	}],
	["borderBottom", {
		w: "borderBottomWidth",
		s: "borderBottomStyle",
		c: "borderBottomColor"
	}],
	["borderLeft", {
		w: "borderLeftWidth",
		s: "borderLeftStyle",
		c: "borderLeftColor"
	}]
]);
function tableBorderParser(format, element) {
	BorderKeys.forEach(function(key) {
		if (!format[key]) {
			var styleSet = ElementBorderKeys.get(key);
			if (styleSet && element.style[styleSet.w] && element.style[styleSet.s] && !element.style[styleSet.c]) format[key] = element.style[styleSet.w] + " " + element.style[styleSet.s];
		}
	});
}
var deleteAllSegmentBefore = function(context) {
	if (context.deleteResult != "notDeleted") return;
	var _a$5 = context.insertPoint, paragraph = _a$5.paragraph, marker = _a$5.marker;
	var index$1 = paragraph.segments.indexOf(marker);
	var mutableParagraph = mutateBlock(paragraph);
	for (var i$1 = index$1 - 1; i$1 >= 0; i$1--) {
		var segment = mutableParagraph.segments[i$1];
		segment.isSelected = true;
		if (deleteSegment(paragraph, segment, context.formatContext)) context.deleteResult = "range";
	}
};
var deleteEmptyQuote = function(context) {
	var deleteResult = context.deleteResult;
	if (deleteResult == "nothingToDelete" || deleteResult == "notDeleted" || deleteResult == "range") {
		var insertPoint = context.insertPoint, formatContext = context.formatContext;
		var path = insertPoint.path, paragraph = insertPoint.paragraph;
		var rawEvent = formatContext === null || formatContext === void 0 ? void 0 : formatContext.rawEvent;
		var index$1 = getClosestAncestorBlockGroupIndex(path, ["FormatContainer"], ["TableCell", "ListItem"]);
		var quote = path[index$1];
		if (quote && quote.blockGroupType === "FormatContainer" && quote.tagName == "blockquote") {
			var parent_1 = path[index$1 + 1];
			var quoteBlockIndex = parent_1.blocks.indexOf(quote);
			if (isEmptyQuote(quote)) {
				unwrapBlock(parent_1, quote);
				rawEvent === null || rawEvent === void 0 || rawEvent.preventDefault();
				context.deleteResult = "range";
			} else if ((rawEvent === null || rawEvent === void 0 ? void 0 : rawEvent.key) === "Enter" && quote.blocks.indexOf(paragraph) >= 0 && isEmptyParagraph$1(paragraph)) {
				insertNewLine(mutateBlock(quote), parent_1, quoteBlockIndex, paragraph);
				rawEvent === null || rawEvent === void 0 || rawEvent.preventDefault();
				context.deleteResult = "range";
			}
		}
	}
};
var isEmptyQuote = function(quote) {
	return quote.blocks.length === 1 && quote.blocks[0].blockType === "Paragraph" && isEmptyParagraph$1(quote.blocks[0]);
};
var isEmptyParagraph$1 = function(paragraph) {
	return paragraph.segments.every(function(s$1) {
		return s$1.segmentType === "SelectionMarker" || s$1.segmentType === "Br";
	});
};
var insertNewLine = function(quote, parent, quoteIndex, paragraph) {
	var _a$5;
	var paraIndex = quote.blocks.indexOf(paragraph);
	if (paraIndex >= 0) {
		var mutableParent = mutateBlock(parent);
		if (paraIndex < quote.blocks.length - 1) {
			var newQuote = createFormatContainer(quote.tagName, quote.format);
			(_a$5 = newQuote.blocks).push.apply(_a$5, __spreadArray([], __read(quote.blocks.splice(paraIndex + 1, quote.blocks.length - paraIndex - 1)), false));
			mutableParent.blocks.splice(quoteIndex + 1, 0, newQuote);
		}
		mutableParent.blocks.splice(quoteIndex + 1, 0, paragraph);
		quote.blocks.splice(paraIndex, 1);
		if (quote.blocks.length == 0) mutableParent.blocks.splice(quoteIndex, 0);
	}
};
var deleteList = function(context) {
	if (context.deleteResult != "notDeleted") return;
	var _a$5 = context.insertPoint, paragraph = _a$5.paragraph, marker = _a$5.marker, path = _a$5.path;
	if (paragraph.segments[0] == marker) {
		var item = path[getClosestAncestorBlockGroupIndex(path, ["ListItem"], ["TableCell", "FormatContainer"])];
		var lastLevel = item === null || item === void 0 ? void 0 : item.levels[item.levels.length - 1];
		if (lastLevel && (item === null || item === void 0 ? void 0 : item.blocks[0]) == paragraph) {
			if (lastLevel.format.displayForDummyItem == "block") item.levels.pop();
			else lastLevel.format.displayForDummyItem = "block";
			context.deleteResult = "range";
		}
	}
};
var deleteParagraphStyle = function(context) {
	if (context.deleteResult === "nothingToDelete") {
		var insertPoint = context.insertPoint;
		var paragraph = insertPoint.paragraph, path = insertPoint.path;
		var group = path[0];
		var parentGroup = path[1];
		if (paragraph.segments.every(function(s$1) {
			return s$1.segmentType === "SelectionMarker" || s$1.segmentType === "Br";
		}) && paragraph.segments.filter(function(s$1) {
			return s$1.segmentType === "Br";
		}).length <= 1) {
			if (Object.keys(paragraph.format).length > 0) {
				paragraph.format = {};
				context.deleteResult = "range";
			} else if (group.blocks.length == 1 && group.blocks[0] == paragraph && parentGroup && (group.blockGroupType == "FormatContainer" || group.blockGroupType == "ListItem" || group.blockGroupType == "General")) {
				unwrapBlock(parentGroup, group);
				path.shift();
				context.deleteResult = "range";
			}
		}
	}
};
function getLeafSiblingBlock(path, block, isNext) {
	var _a$5;
	var newPath = __spreadArray([], __read(path), false);
	var _loop_1 = function() {
		var group = newPath[0];
		var index$1 = group.blocks.indexOf(block);
		if (index$1 < 0) return "break";
		var nextBlock = group.blocks[index$1 + (isNext ? 1 : -1)];
		if (nextBlock) {
			while (nextBlock.blockType == "BlockGroup") {
				var child$1 = nextBlock.blocks[isNext ? 0 : nextBlock.blocks.length - 1];
				if (!child$1) return { value: {
					block: nextBlock,
					path: newPath
				} };
				else if (child$1.blockType != "BlockGroup") {
					newPath.unshift(nextBlock);
					return { value: {
						block: child$1,
						path: newPath
					} };
				} else {
					newPath.unshift(nextBlock);
					nextBlock = child$1;
				}
			}
			return { value: {
				block: nextBlock,
				path: newPath
			} };
		} else if (isGeneralSegment(group)) {
			newPath.shift();
			var segmentIndex_1 = -1;
			var segment_1 = group;
			var para = (_a$5 = newPath[0]) === null || _a$5 === void 0 ? void 0 : _a$5.blocks.find(function(x$1) {
				return x$1.blockType == "Paragraph" && (segmentIndex_1 = x$1.segments.indexOf(segment_1)) >= 0;
			});
			if (para) {
				var siblingSegment = para.segments[segmentIndex_1 + (isNext ? 1 : -1)];
				if (siblingSegment) return { value: {
					block: para,
					path: newPath,
					siblingSegment
				} };
				else block = para;
			} else return "break";
		} else if (group.blockGroupType != "Document" && group.blockGroupType != "TableCell") {
			newPath.shift();
			block = group;
		} else return "break";
	};
	while (newPath.length > 0) {
		var state_1 = _loop_1();
		if (typeof state_1 === "object") return state_1.value;
		if (state_1 === "break") break;
	}
	return null;
}
function preserveParagraphFormat(formatsToPreserveOnMerge, paragraph, newParagraph) {
	if (formatsToPreserveOnMerge && formatsToPreserveOnMerge.length) {
		var format_1 = paragraph.format;
		var newFormat_1 = newParagraph.format;
		formatsToPreserveOnMerge.forEach(function(key) {
			var formatValue = format_1[key];
			if (formatValue !== void 0) newFormat_1[key] = formatValue;
		});
	}
}
function getDeleteCollapsedSelection(direction, options) {
	return function(context) {
		var _a$5;
		if (context.deleteResult != "notDeleted") return;
		var isForward = direction == "forward";
		var _b$1 = context.insertPoint, paragraph = _b$1.paragraph, marker = _b$1.marker, path = _b$1.path, tableContext = _b$1.tableContext;
		var segments = paragraph.segments;
		fixupBr(paragraph);
		var segmentToDelete = segments[segments.indexOf(marker) + (isForward ? 1 : -1)];
		var blockToDelete;
		var root$12;
		if (segmentToDelete) {
			if (deleteSegment(paragraph, segmentToDelete, context.formatContext, direction)) {
				context.deleteResult = "singleChar";
				setParagraphNotImplicit(paragraph);
			}
		} else if (shouldOutdentParagraph(isForward, segments, paragraph, path) && (root$12 = getRoot(path))) {
			setModelIndentation(root$12, "outdent");
			context.deleteResult = "range";
		} else if (blockToDelete = getLeafSiblingBlock(path, paragraph, isForward)) {
			var readonlyBlock = blockToDelete.block, path_1 = blockToDelete.path, siblingSegment = blockToDelete.siblingSegment;
			if (readonlyBlock.blockType == "Paragraph") {
				var block = mutateBlock(readonlyBlock);
				if (siblingSegment) {
					if (deleteSegment(block, siblingSegment, context.formatContext, direction)) context.deleteResult = "range";
				} else {
					if (isForward) context.lastParagraph = block;
					else {
						if (((_a$5 = block.segments[block.segments.length - 1]) === null || _a$5 === void 0 ? void 0 : _a$5.segmentType) == "Br") mutateBlock(block).segments.pop();
						context.insertPoint = {
							marker,
							paragraph: block,
							path: path_1,
							tableContext
						};
						context.lastParagraph = paragraph;
					}
					preserveParagraphFormat(options.formatsToPreserveOnMerge, context.insertPoint.paragraph, context.lastParagraph);
					context.deleteResult = "range";
				}
				context.lastTableContext = tableContext;
			} else if (deleteBlock(mutateBlock(path_1[0]).blocks, readonlyBlock, void 0, context.formatContext, direction)) context.deleteResult = "range";
		} else context.deleteResult = "nothingToDelete";
	};
}
function getRoot(path) {
	var lastInPath = path[path.length - 1];
	return lastInPath.blockGroupType == "Document" ? lastInPath : null;
}
function shouldOutdentParagraph(isForward, segments, paragraph, path) {
	return !isForward && segments.length == 1 && segments[0].segmentType == "SelectionMarker" && paragraph.format.marginLeft && parseInt(paragraph.format.marginLeft) && getClosestAncestorBlockGroupIndex(path, ["Document", "TableCell"], ["ListItem"]) > -1;
}
function fixupBr(paragraph) {
	var _a$5, _b$1;
	var segments = paragraph.segments;
	if (((_a$5 = segments[segments.length - 1]) === null || _a$5 === void 0 ? void 0 : _a$5.segmentType) == "Br") {
		var segmentsWithoutBr = segments.filter(function(x$1) {
			return x$1.segmentType != "SelectionMarker";
		});
		if (((_b$1 = segmentsWithoutBr[segmentsWithoutBr.length - 2]) === null || _b$1 === void 0 ? void 0 : _b$1.segmentType) != "Br") mutateBlock(paragraph).segments.pop();
	}
}
function handleKeyboardEventResult(editor, model, rawEvent, result, context) {
	context.skipUndoSnapshot = true;
	context.clearModelCache = false;
	switch (result) {
		case "notDeleted":
			context.clearModelCache = true;
			return false;
		case "nothingToDelete":
			rawEvent.preventDefault();
			return false;
		case "range":
		case "singleChar":
			rawEvent.preventDefault();
			normalizeContentModel(model);
			deleteEmptyBlockGroups(model);
			if (result == "range") context.skipUndoSnapshot = false;
			editor.triggerEvent("beforeKeyboardEditing", { rawEvent });
			return true;
	}
}
function shouldDeleteWord(rawEvent, isMac) {
	return isMac && rawEvent.altKey && !rawEvent.metaKey || !isMac && rawEvent.ctrlKey && !rawEvent.altKey;
}
function shouldDeleteAllSegmentsBefore(rawEvent) {
	return rawEvent.metaKey && !rawEvent.altKey;
}
function deleteEmptyBlockGroups(group) {
	var modified = false;
	for (var i$1 = group.blocks.length - 1; i$1 >= 0; i$1--) {
		var block = group.blocks[i$1];
		if (block.blockType == "BlockGroup") {
			deleteEmptyBlockGroups(block);
			if (block.blocks.length == 0) {
				mutateBlock(group).blocks.splice(i$1, 1);
				modified = true;
			}
		}
	}
	if (modified) group.blocks.forEach(setParagraphNotImplicit);
}
var DeleteWordState;
(function(DeleteWordState$1) {
	DeleteWordState$1[DeleteWordState$1["Start"] = 0] = "Start";
	DeleteWordState$1[DeleteWordState$1["Punctuation"] = 1] = "Punctuation";
	DeleteWordState$1[DeleteWordState$1["Text"] = 2] = "Text";
	DeleteWordState$1[DeleteWordState$1["NonText"] = 3] = "NonText";
	DeleteWordState$1[DeleteWordState$1["Space"] = 4] = "Space";
	DeleteWordState$1[DeleteWordState$1["End"] = 5] = "End";
})(DeleteWordState || (DeleteWordState = {}));
function getDeleteWordSelection(direction) {
	return function(context) {
		if (context.deleteResult != "notDeleted") return;
		var _a$5 = context.insertPoint, marker = _a$5.marker, paragraph = _a$5.paragraph;
		var startIndex = paragraph.segments.indexOf(marker);
		var deleteNext = direction == "forward";
		var iterator = iterateSegments(mutateBlock(paragraph), startIndex, deleteNext, context);
		var curr = iterator.next();
		for (var state$1 = 0; state$1 != 5 && !curr.done;) {
			var _b$1 = curr.value, punctuation = _b$1.punctuation, space$1 = _b$1.space, text = _b$1.text;
			switch (state$1) {
				case 0:
					state$1 = space$1 ? 4 : punctuation ? 1 : 2;
					curr = iterator.next(true);
					break;
				case 1:
					if (deleteNext && space$1) {
						state$1 = 3;
						curr = iterator.next(true);
					} else if (punctuation) curr = iterator.next(true);
					else state$1 = 5;
					break;
				case 2:
					if (deleteNext && space$1) {
						state$1 = 3;
						curr = iterator.next(true);
					} else if (text) curr = iterator.next(true);
					else state$1 = 5;
					break;
				case 3:
					if (punctuation || !space$1) state$1 = 5;
					else curr = iterator.next(true);
					break;
				case 4:
					if (space$1) curr = iterator.next(true);
					else if (punctuation) {
						state$1 = deleteNext ? 3 : 1;
						curr = iterator.next(true);
					} else state$1 = deleteNext ? 5 : 2;
					break;
			}
		}
	};
}
function iterateSegments(paragraph, markerIndex, forward, context) {
	var step, segments, preserveWhiteSpace, i$1, segment, _a$5, j$1, c$1, punctuation, space$1, text, newText;
	return __generator(this, function(_b$1) {
		switch (_b$1.label) {
			case 0:
				step = forward ? 1 : -1;
				segments = paragraph.segments;
				preserveWhiteSpace = isWhiteSpacePreserved(paragraph.format.whiteSpace);
				i$1 = markerIndex + step;
				_b$1.label = 1;
			case 1:
				if (!(i$1 >= 0 && i$1 < segments.length)) return [3, 12];
				segment = segments[i$1];
				_a$5 = segment.segmentType;
				switch (_a$5) {
					case "Text": return [3, 2];
					case "Image": return [3, 7];
					case "SelectionMarker": return [3, 9];
				}
				return [3, 10];
			case 2:
				j$1 = forward ? 0 : segment.text.length - 1;
				_b$1.label = 3;
			case 3:
				if (!(j$1 >= 0 && j$1 < segment.text.length)) return [3, 6];
				c$1 = segment.text[j$1];
				punctuation = isPunctuation(c$1);
				space$1 = isSpace(c$1);
				text = !punctuation && !space$1;
				return [4, {
					punctuation,
					space: space$1,
					text
				}];
			case 4:
				if (_b$1.sent()) {
					newText = segment.text;
					newText = newText.substring(0, j$1) + newText.substring(j$1 + 1);
					if (!preserveWhiteSpace) newText = normalizeText(newText, forward);
					context.deleteResult = "range";
					if (newText) {
						segment.text = newText;
						if (step > 0) j$1 -= step;
					} else {
						segments.splice(i$1, 1);
						if (step > 0) i$1 -= step;
						return [3, 6];
					}
				}
				_b$1.label = 5;
			case 5:
				j$1 += step;
				return [3, 3];
			case 6: return [3, 11];
			case 7: return [4, {
				punctuation: true,
				space: false,
				text: false
			}];
			case 8:
				if (_b$1.sent()) {
					segments.splice(i$1, 1);
					if (step > 0) i$1 -= step;
					context.deleteResult = "range";
				}
				return [3, 11];
			case 9: return [3, 11];
			case 10: return [2, null];
			case 11:
				i$1 += step;
				return [3, 1];
			case 12: return [2, null];
		}
	});
}
var forwardDeleteWordSelection = getDeleteWordSelection("forward");
var backwardDeleteWordSelection = getDeleteWordSelection("backward");
function keyboardDelete(editor, rawEvent, options) {
	var handled = false;
	var selection = editor.getDOMSelection();
	var handleExpandedSelectionOnDelete = options.handleExpandedSelectionOnDelete;
	if (shouldDeleteWithContentModel(selection, rawEvent, !!handleExpandedSelectionOnDelete)) editor.formatContentModel(function(model, context) {
		var result = deleteSelection(model, getDeleteSteps(rawEvent, !!editor.getEnvironment().isMac, options), context).deleteResult;
		handled = handleKeyboardEventResult(editor, model, rawEvent, result, context);
		return handled;
	}, {
		rawEvent,
		changeSource: ChangeSource.Keyboard,
		getChangeData: function() {
			return rawEvent.which;
		},
		scrollCaretIntoView: true,
		apiName: rawEvent.key == "Delete" ? "handleDeleteKey" : "handleBackspaceKey"
	});
	return handled;
}
function getDeleteSteps(rawEvent, isMac, options) {
	var isForward = rawEvent.key == "Delete";
	var deleteAllSegmentBeforeStep = shouldDeleteAllSegmentsBefore(rawEvent) && !isForward ? deleteAllSegmentBefore : null;
	var deleteWordSelection = shouldDeleteWord(rawEvent, isMac) ? isForward ? forwardDeleteWordSelection : backwardDeleteWordSelection : null;
	var deleteCollapsedSelection = getDeleteCollapsedSelection(isForward ? "forward" : "backward", options);
	return [
		deleteAllSegmentBeforeStep,
		deleteWordSelection,
		isForward ? null : deleteList,
		deleteCollapsedSelection,
		!isForward ? deleteEmptyQuote : null,
		deleteParagraphStyle
	];
}
function shouldDeleteWithContentModel(selection, rawEvent, handleExpandedSelection) {
	var _a$5, _b$1;
	if (!selection) return false;
	else if (selection.type != "range") return true;
	else if (!selection.range.collapsed) {
		if (handleExpandedSelection) return true;
		var range = selection.range;
		var _c$1 = selection.range, startContainer = _c$1.startContainer;
		return !(startContainer === _c$1.endContainer && isNodeOfType(startContainer, "TEXT_NODE") && !isModifierKey(rawEvent) && range.endOffset - range.startOffset < ((_b$1 = (_a$5 = startContainer.nodeValue) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0));
	} else {
		var range = selection.range;
		var startContainer = range.startContainer;
		var startOffset = range.startOffset;
		return !(isNodeOfType(startContainer, "TEXT_NODE") && !isModifierKey(rawEvent) && (canDeleteBefore(rawEvent, startContainer, startOffset) || canDeleteAfter(rawEvent, startContainer, startOffset)));
	}
}
function canDeleteBefore(rawEvent, text, offset) {
	var _a$5, _b$1;
	if (rawEvent.key != "Backspace" || offset <= 1) return false;
	if (offset == ((_b$1 = (_a$5 = text.nodeValue) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0)) {
		var nextSibling = text.nextSibling;
		return !(isNodeOfType(nextSibling, "ELEMENT_NODE") && isElementOfType(nextSibling, "a") && isLinkUndeletable(nextSibling) && !nextSibling.firstChild);
	} else return true;
}
function canDeleteAfter(rawEvent, text, offset) {
	var _a$5, _b$1;
	return rawEvent.key == "Delete" && offset < ((_b$1 = (_a$5 = text.nodeValue) === null || _a$5 === void 0 ? void 0 : _a$5.length) !== null && _b$1 !== void 0 ? _b$1 : 0) - 1;
}
var handleAutoLink = function(context) {
	var deleteResult = context.deleteResult, insertPoint = context.insertPoint;
	if (deleteResult == "notDeleted" || deleteResult == "nothingToDelete") {
		var marker = insertPoint.marker, paragraph = insertPoint.paragraph;
		var index$1 = paragraph.segments.indexOf(marker);
		var segBefore = index$1 > 0 ? paragraph.segments[index$1 - 1] : null;
		if ((segBefore === null || segBefore === void 0 ? void 0 : segBefore.segmentType) == "Text" && promoteLink(segBefore, paragraph, { autoLink: true }) && context.formatContext) context.formatContext.canUndoByBackspace = true;
	}
};
function splitParagraph(insertPoint, removeImplicitParagraph, formatsToPreserveOnMerge) {
	var _a$5;
	if (removeImplicitParagraph === void 0) removeImplicitParagraph = true;
	if (formatsToPreserveOnMerge === void 0) formatsToPreserveOnMerge = [];
	var paragraph = insertPoint.paragraph, marker = insertPoint.marker;
	var newParagraph = createParagraph(false, {}, paragraph.segmentFormat);
	copyFormat(newParagraph.format, paragraph.format, ParagraphFormats);
	preserveParagraphFormat(formatsToPreserveOnMerge, paragraph, newParagraph);
	var markerIndex = paragraph.segments.indexOf(marker);
	var segments = paragraph.segments.splice(markerIndex, paragraph.segments.length - markerIndex);
	(_a$5 = newParagraph.segments).push.apply(_a$5, __spreadArray([], __read(segments), false));
	var isEmptyParagraph$2 = paragraph.segments.length == 0;
	var shouldPreserveImplicitParagraph = !paragraph.isImplicit || !removeImplicitParagraph;
	if (isEmptyParagraph$2 && shouldPreserveImplicitParagraph) paragraph.segments.push(createBr(marker.format));
	else if (!isEmptyParagraph$2) setParagraphNotImplicit(paragraph);
	insertPoint.paragraph = newParagraph;
	normalizeParagraph(paragraph);
	return newParagraph;
}
var handleEnterOnList = function(context) {
	var deleteResult = context.deleteResult, insertPoint = context.insertPoint;
	if (deleteResult == "notDeleted" || deleteResult == "nothingToDelete") {
		var path = insertPoint.path;
		var index$1 = getClosestAncestorBlockGroupIndex(path, ["ListItem"], ["TableCell", "FormatContainer"]);
		var readonlyListItem = path[index$1];
		var listParent = path[index$1 + 1];
		if ((readonlyListItem === null || readonlyListItem === void 0 ? void 0 : readonlyListItem.blockGroupType) === "ListItem" && listParent) {
			var listItem = mutateBlock(readonlyListItem);
			if (isEmptyListItem(listItem)) listItem.levels.pop();
			else {
				listItem = createNewListItem(context, listItem, listParent);
				if (context.formatContext) context.formatContext.announceData = getListAnnounceData(__spreadArray([listItem], __read(path.slice(index$1 + 1)), false));
			}
			var listIndex = listParent.blocks.indexOf(listItem);
			var nextBlock = listParent.blocks[listIndex + 1];
			if (nextBlock) {
				if (isBlockGroupOfType(nextBlock, "ListItem") && nextBlock.levels[0]) {
					nextBlock.levels.forEach(function(level) {
						if (level.format.startNumberOverride !== 1) level.format.startNumberOverride = void 0;
					});
					if (listItem.levels.length == 0) {
						var nextBlockIndex = findIndex(listParent.blocks, nextBlock.levels.length);
						nextBlock.levels[nextBlock.levels.length - 1].format.startNumberOverride = nextBlockIndex;
					}
				}
			}
			context.deleteResult = "range";
		}
	}
};
var isEmptyListItem = function(listItem) {
	return listItem.blocks.length === 1 && isEmptyParagraph(listItem.blocks[0]);
};
var isEmptyParagraph = function(block) {
	return block.blockType === "Paragraph" && block.segments.length === 2 && block.segments[0].segmentType === "SelectionMarker" && block.segments[1].segmentType === "Br";
};
var createNewListItem = function(context, listItem, listParent) {
	var _a$5;
	var insertPoint = context.insertPoint;
	var listIndex = listParent.blocks.indexOf(listItem);
	var currentPara = insertPoint.paragraph;
	var paraIndex = listItem.blocks.indexOf(currentPara);
	var newParagraph = splitParagraph(insertPoint);
	var newListItem = createListItem(createNewListLevel(listItem), listItem.formatHolder.format);
	newListItem.blocks.push(newParagraph);
	copyFormat(newListItem.format, listItem.format, ListFormats);
	var remainingBlockCount = listItem.blocks.length - paraIndex - 1;
	if (paraIndex >= 0 && remainingBlockCount > 0) (_a$5 = newListItem.blocks).push.apply(_a$5, __spreadArray([], __read(mutateBlock(listItem).blocks.splice(paraIndex + 1, remainingBlockCount)), false));
	insertPoint.paragraph = newParagraph;
	mutateBlock(listParent).blocks.splice(listIndex + 1, 0, newListItem);
	if (context.lastParagraph == currentPara) context.lastParagraph = newParagraph;
	return newListItem;
};
var createNewListLevel = function(listItem) {
	return listItem.levels.map(function(level) {
		return createListLevel(level.listType, __assign(__assign({}, level.format), {
			startNumberOverride: void 0,
			displayForDummyItem: void 0
		}), level.dataset);
	});
};
var findIndex = function(blocks, levelLength) {
	var counter = 1;
	for (var i$1 = 0; i$1 < blocks.length; i$1++) {
		var listItem = blocks[i$1];
		if (isBlockGroupOfType(listItem, "ListItem") && listItem.levels.length === levelLength) counter++;
		else if (isBlockGroupOfType(listItem, "ListItem") && listItem.levels.length == 0) return counter;
	}
	return counter;
};
var handleEnterOnParagraph = function(formatsToPreserveOnMerge) {
	return function(context) {
		var _a$5, _b$1;
		var _c$1 = context.insertPoint, paragraph = _c$1.paragraph, path = _c$1.path;
		var paraIndex = (_b$1 = (_a$5 = path[0]) === null || _a$5 === void 0 ? void 0 : _a$5.blocks.indexOf(paragraph)) !== null && _b$1 !== void 0 ? _b$1 : -1;
		if (context.deleteResult == "notDeleted" && paraIndex >= 0) {
			var newPara = splitParagraph(context.insertPoint, false, formatsToPreserveOnMerge);
			mutateBlock(path[0]).blocks.splice(paraIndex + 1, 0, newPara);
			context.deleteResult = "range";
			context.lastParagraph = newPara;
			context.insertPoint.paragraph = newPara;
		}
	};
};
function keyboardEnter(editor, rawEvent, handleNormalEnter, formatsToPreserveOnMerge) {
	if (formatsToPreserveOnMerge === void 0) formatsToPreserveOnMerge = [];
	var selection = editor.getDOMSelection();
	editor.formatContentModel(function(model, context) {
		var _a$5, _b$1;
		var result = deleteSelection(model, [], context);
		if (selection && selection.type != "table") {
			result.deleteResult = "notDeleted";
			var steps = rawEvent.shiftKey ? [] : [
				handleAutoLink,
				handleEnterOnList,
				deleteEmptyQuote
			];
			if (handleNormalEnter || handleEnterForEntity((_a$5 = result.insertPoint) === null || _a$5 === void 0 ? void 0 : _a$5.paragraph)) steps.push(handleEnterOnParagraph(formatsToPreserveOnMerge));
			runEditSteps(steps, result);
		}
		if (result.deleteResult == "range") {
			context.newPendingFormat = (_b$1 = result.insertPoint) === null || _b$1 === void 0 ? void 0 : _b$1.marker.format;
			normalizeContentModel(model);
			rawEvent.preventDefault();
			return true;
		} else return false;
	}, {
		rawEvent,
		scrollCaretIntoView: true,
		changeSource: ChangeSource.Keyboard,
		getChangeData: function() {
			return rawEvent.which;
		},
		apiName: "handleEnterKey"
	});
}
function handleEnterForEntity(paragraph) {
	return paragraph && (paragraph.isImplicit || paragraph.segments.some(function(x$1) {
		return x$1.segmentType == "Entity";
	}));
}
var ZWS = "​";
var insertZWS = function(context) {
	if (context.deleteResult == "range") {
		var _a$5 = context.insertPoint, marker = _a$5.marker, paragraph = _a$5.paragraph;
		var index$1 = paragraph.segments.indexOf(marker);
		if (index$1 >= 0) {
			var text = createText(ZWS, marker.format, marker.link, marker.code);
			text.isSelected = true;
			paragraph.segments.splice(index$1, 0, text);
		}
	}
};
function keyboardInput(editor, rawEvent) {
	if (shouldInputWithContentModel(editor.getDOMSelection(), rawEvent)) {
		editor.takeSnapshot();
		editor.formatContentModel(function(model, context) {
			var _a$5;
			var result = deleteSelection(model, [insertZWS], context);
			context.skipUndoSnapshot = true;
			if (result.deleteResult == "range") {
				context.newPendingFormat = (_a$5 = result.insertPoint) === null || _a$5 === void 0 ? void 0 : _a$5.marker.format;
				normalizeContentModel(model);
				return true;
			} else return false;
		}, {
			scrollCaretIntoView: true,
			rawEvent,
			changeSource: ChangeSource.Keyboard,
			getChangeData: function() {
				return rawEvent.which;
			},
			apiName: "handleInputKey"
		});
		return true;
	}
}
function shouldInputWithContentModel(selection, rawEvent) {
	if (!selection) return false;
	else if (!isModifierKey(rawEvent) && rawEvent.key && (rawEvent.key == "Space" || rawEvent.key.length == 1)) return selection.type != "range" || !selection.range.collapsed;
	else return false;
}
var tabSpaces = "    ";
var space = " ";
function handleTabOnParagraph(model, paragraph, rawEvent, context) {
	var selectedSegments = paragraph.segments.filter(function(segment) {
		return segment.isSelected;
	});
	var isCollapsed = selectedSegments.length === 1 && selectedSegments[0].segmentType === "SelectionMarker";
	if (paragraph.segments.every(function(segment) {
		return segment.isSelected || segment.segmentType == "Text" && segment.text.trim().length == 0;
	})) {
		var _a$5 = paragraph.format, marginLeft = _a$5.marginLeft, marginRight = _a$5.marginRight;
		var isRtl = _a$5.direction === "rtl";
		if (rawEvent.shiftKey && (!isRtl && (!marginLeft || marginLeft == "0px") || isRtl && (!marginRight || marginRight == "0px"))) return false;
		setModelIndentation(model, rawEvent.shiftKey ? "outdent" : "indent", void 0, context);
	} else if (!isCollapsed) {
		var firstSelectedSegmentIndex_1 = void 0;
		var lastSelectedSegmentIndex_1 = void 0;
		paragraph.segments.forEach(function(segment, index$1) {
			if (segment.isSelected) {
				if (!firstSelectedSegmentIndex_1) firstSelectedSegmentIndex_1 = index$1;
				lastSelectedSegmentIndex_1 = index$1;
			}
		});
		if (firstSelectedSegmentIndex_1 && lastSelectedSegmentIndex_1) {
			var firstSelectedSegment = paragraph.segments[firstSelectedSegmentIndex_1];
			var spaceText = createText(rawEvent.shiftKey ? tabSpaces : space, firstSelectedSegment.format);
			var marker = createSelectionMarker(firstSelectedSegment.format);
			mutateBlock(paragraph).segments.splice(firstSelectedSegmentIndex_1, lastSelectedSegmentIndex_1 - firstSelectedSegmentIndex_1 + 1, spaceText, marker);
		} else return false;
	} else {
		var markerIndex = paragraph.segments.findIndex(function(segment) {
			return segment.segmentType === "SelectionMarker";
		});
		if (!rawEvent.shiftKey) {
			var markerFormat = paragraph.segments[markerIndex].format;
			var tabText = createText(tabSpaces, markerFormat);
			mutateBlock(paragraph).segments.splice(markerIndex, 0, tabText);
		} else {
			if (markerIndex <= 0) return false;
			var tabText = paragraph.segments[markerIndex - 1];
			var tabSpacesLength = tabSpaces.length;
			if (tabText.segmentType == "Text") {
				var tabSpaceTextLength_1 = tabText.text.length - tabSpacesLength;
				if (tabText.text === tabSpaces) mutateBlock(paragraph).segments.splice(markerIndex - 1, 1);
				else if (tabText.text.substring(tabSpaceTextLength_1) === tabSpaces) mutateSegment(paragraph, tabText, function(text) {
					text.text = text.text.substring(0, tabSpaceTextLength_1);
				});
				else return false;
			}
		}
	}
	rawEvent.preventDefault();
	return true;
}
function handleTabOnList(model, listItem, rawEvent, context) {
	var selectedParagraph = findSelectedParagraph(listItem);
	if (!isMarkerAtStartOfBlock(listItem) && selectedParagraph.length == 1 && selectedParagraph[0].blockType === "Paragraph") return handleTabOnParagraph(model, selectedParagraph[0], rawEvent, context);
	else {
		setModelIndentation(model, rawEvent.shiftKey ? "outdent" : "indent", void 0, context);
		rawEvent.preventDefault();
		return true;
	}
}
function isMarkerAtStartOfBlock(listItem) {
	return listItem.blocks[0].blockType == "Paragraph" && listItem.blocks[0].segments[0].segmentType == "SelectionMarker";
}
function findSelectedParagraph(listItem) {
	return listItem.blocks.filter(function(block) {
		return block.blockType == "Paragraph" && block.segments.some(function(segment) {
			return segment.isSelected;
		});
	});
}
function handleTabOnTable(model, rawEvent) {
	var tableModel = getFirstSelectedTable(model)[0];
	if (tableModel && isWholeTableSelected(tableModel)) {
		setModelIndentation(model, rawEvent.shiftKey ? "outdent" : "indent");
		rawEvent.preventDefault();
		return true;
	}
	return false;
}
function isWholeTableSelected(tableModel) {
	var _a$5, _b$1;
	var lastRow = tableModel.rows[tableModel.rows.length - 1];
	var lastCell = lastRow === null || lastRow === void 0 ? void 0 : lastRow.cells[lastRow.cells.length - 1];
	return ((_b$1 = (_a$5 = tableModel.rows[0]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[0]) === null || _b$1 === void 0 ? void 0 : _b$1.isSelected) && (lastCell === null || lastCell === void 0 ? void 0 : lastCell.isSelected);
}
function handleTabOnTableCell(model, cell, rawEvent) {
	var _a$5, _b$1;
	var readonlyTableModel = getFirstSelectedTable(model)[0];
	if (readonlyTableModel) {
		var lastRow = readonlyTableModel.rows[readonlyTableModel.rows.length - 1];
		var lastColumn = lastRow ? lastRow.cells.length - 1 : -1;
		var lastCell = lastRow === null || lastRow === void 0 ? void 0 : lastRow.cells[lastColumn];
		if (!rawEvent.shiftKey && lastCell && lastCell === cell) {
			var tableModel = mutateBlock(readonlyTableModel);
			insertTableRow(tableModel, "insertBelow");
			clearSelectedCells(tableModel, {
				firstRow: tableModel.rows.length - 1,
				firstColumn: 0,
				lastRow: tableModel.rows.length - 1,
				lastColumn
			});
			normalizeTable$1(tableModel, model.format);
			var markerParagraph = (_b$1 = (_a$5 = tableModel.rows[tableModel.rows.length - 1]) === null || _a$5 === void 0 ? void 0 : _a$5.cells[0]) === null || _b$1 === void 0 ? void 0 : _b$1.blocks[0];
			if (markerParagraph.blockType == "Paragraph") {
				var marker = createSelectionMarker(model.format);
				mutateBlock(markerParagraph).segments.unshift(marker);
				setParagraphNotImplicit(markerParagraph);
				setSelection(tableModel.rows[tableModel.rows.length - 1].cells[0], marker);
			}
			rawEvent.preventDefault();
			return true;
		}
	}
	return false;
}
function keyboardTab(editor, rawEvent) {
	var selection = editor.getDOMSelection();
	switch (selection === null || selection === void 0 ? void 0 : selection.type) {
		case "range":
			editor.formatContentModel(function(model, context) {
				return handleTab(model, rawEvent, context);
			}, {
				apiName: "handleTabKey",
				rawEvent,
				changeSource: ChangeSource.Keyboard,
				getChangeData: function() {
					return rawEvent.which;
				}
			});
			return true;
		case "table":
			editor.formatContentModel(function(model) {
				return handleTabOnTable(model, rawEvent);
			}, {
				apiName: "handleTabKey",
				rawEvent,
				changeSource: ChangeSource.Keyboard,
				getChangeData: function() {
					return rawEvent.which;
				}
			});
			return true;
	}
}
function handleTab(model, rawEvent, context) {
	var blocks = getOperationalBlocks(model, ["ListItem", "TableCell"], []);
	var block = blocks.length > 0 ? blocks[0].block : void 0;
	if (blocks.length > 1) {
		setModelIndentation(model, rawEvent.shiftKey ? "outdent" : "indent");
		rawEvent.preventDefault();
		return true;
	} else if (isBlockGroupOfType(block, "TableCell")) return handleTabOnTableCell(model, block, rawEvent);
	else if ((block === null || block === void 0 ? void 0 : block.blockType) === "Paragraph") return handleTabOnParagraph(model, block, rawEvent, context);
	else if (isBlockGroupOfType(block, "ListItem")) return handleTabOnList(model, block, rawEvent, context);
	return false;
}
var BACKSPACE_KEY = 8;
var DELETE_KEY = 46;
var DEAD_KEY = 229;
var DefaultOptions$3 = {
	handleTabKey: true,
	handleExpandedSelectionOnDelete: true
};
var EditPlugin = function() {
	function EditPlugin$1(options) {
		if (options === void 0) options = DefaultOptions$3;
		this.options = options;
		this.editor = null;
		this.disposer = null;
		this.shouldHandleNextInputEvent = false;
		this.selectionAfterDelete = null;
		this.handleNormalEnter = function() {
			return false;
		};
		this.options = __assign(__assign({}, DefaultOptions$3), options);
	}
	EditPlugin$1.prototype.createNormalEnterChecker = function(result) {
		return result ? function() {
			return true;
		} : function() {
			return false;
		};
	};
	EditPlugin$1.prototype.getHandleNormalEnter = function(editor) {
		switch (typeof this.options.shouldHandleEnterKey) {
			case "function": return this.options.shouldHandleEnterKey;
			case "boolean": return this.createNormalEnterChecker(this.options.shouldHandleEnterKey);
			default: return this.createNormalEnterChecker(editor.isExperimentalFeatureEnabled("HandleEnterKey"));
		}
	};
	EditPlugin$1.prototype.getName = function() {
		return "Edit";
	};
	EditPlugin$1.prototype.initialize = function(editor) {
		var _this = this;
		this.editor = editor;
		this.handleNormalEnter = this.getHandleNormalEnter(editor);
		if (editor.getEnvironment().isAndroid) this.disposer = this.editor.attachDomEvent({ beforeinput: { beforeDispatch: function(e$1) {
			return _this.handleBeforeInputEvent(editor, e$1);
		} } });
	};
	EditPlugin$1.prototype.dispose = function() {
		var _a$5;
		this.editor = null;
		(_a$5 = this.disposer) === null || _a$5 === void 0 || _a$5.call(this);
		this.disposer = null;
	};
	EditPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (this.editor) switch (event$1.eventType) {
			case "keyDown":
				this.handleKeyDownEvent(this.editor, event$1);
				break;
			case "keyUp":
				if (this.selectionAfterDelete) {
					this.editor.setDOMSelection(this.selectionAfterDelete);
					this.selectionAfterDelete = null;
				}
				break;
		}
	};
	EditPlugin$1.prototype.willHandleEventExclusively = function(event$1) {
		if (this.editor && this.options.handleTabKey && event$1.eventType == "keyDown" && event$1.rawEvent.key == "Tab" && !event$1.rawEvent.shiftKey) {
			var selection = this.editor.getDOMSelection();
			var startContainer = (selection === null || selection === void 0 ? void 0 : selection.type) == "range" ? selection.range.startContainer : null;
			var table = startContainer ? this.editor.getDOMHelper().findClosestElementAncestor(startContainer, "table") : null;
			var parsedTable = table && parseTableCells(table);
			if (parsedTable) {
				var lastRow = parsedTable[parsedTable.length - 1];
				var lastCell = lastRow && lastRow[lastRow.length - 1];
				if (typeof lastCell == "object" && lastCell.contains(startContainer)) return true;
			}
		}
		return false;
	};
	EditPlugin$1.prototype.handleKeyDownEvent = function(editor, event$1) {
		var rawEvent = event$1.rawEvent;
		var hasCtrlOrMetaKey = rawEvent.ctrlKey || rawEvent.metaKey;
		if (!rawEvent.defaultPrevented && !event$1.handledByEditFeature) switch (rawEvent.key) {
			case "Backspace":
				if (!this.shouldBrowserHandleBackspace(editor)) keyboardDelete(editor, rawEvent, this.options);
				break;
			case "Delete":
				if (!event$1.rawEvent.shiftKey) keyboardDelete(editor, rawEvent, this.options);
				break;
			case "Tab":
				if (this.options.handleTabKey && !hasCtrlOrMetaKey) keyboardTab(editor, rawEvent);
				break;
			case "Unidentified":
				if (editor.getEnvironment().isAndroid) this.shouldHandleNextInputEvent = true;
				break;
			case "Enter":
				if (!hasCtrlOrMetaKey && !event$1.rawEvent.isComposing && event$1.rawEvent.keyCode !== DEAD_KEY) keyboardEnter(editor, rawEvent, this.handleNormalEnter(editor), this.options.formatsToPreserveOnMerge);
				break;
			default:
				keyboardInput(editor, rawEvent);
				break;
		}
	};
	EditPlugin$1.prototype.handleBeforeInputEvent = function(editor, rawEvent) {
		if (!this.shouldHandleNextInputEvent || !(rawEvent instanceof InputEvent) || rawEvent.defaultPrevented) return;
		this.shouldHandleNextInputEvent = false;
		var handled = false;
		switch (rawEvent.inputType) {
			case "deleteContentBackward":
				if (!this.shouldBrowserHandleBackspace(editor)) handled = keyboardDelete(editor, new KeyboardEvent("keydown", {
					key: "Backspace",
					keyCode: BACKSPACE_KEY,
					which: BACKSPACE_KEY
				}), this.options);
				break;
			case "deleteContentForward":
				handled = keyboardDelete(editor, new KeyboardEvent("keydown", {
					key: "Delete",
					keyCode: DELETE_KEY,
					which: DELETE_KEY
				}), this.options);
				break;
		}
		if (handled) {
			rawEvent.preventDefault();
			this.selectionAfterDelete = editor.getDOMSelection();
		}
	};
	EditPlugin$1.prototype.shouldBrowserHandleBackspace = function(editor) {
		var opt = this.options.shouldHandleBackspaceKey;
		switch (typeof opt) {
			case "function": return opt(editor);
			case "boolean": return opt;
			default: return false;
		}
	};
	return EditPlugin$1;
}();
var HorizontalLineTriggerCharacters = [
	"-",
	"=",
	"_",
	"*",
	"~",
	"#"
];
var commonStyles = {
	width: "98%",
	display: "inline-block"
};
var HorizontalLineStyles = new Map([
	["-", __assign({
		borderTop: "1px none",
		borderRight: "1px none",
		borderBottom: "1px solid",
		borderLeft: "1px none"
	}, commonStyles)],
	["=", __assign({
		borderTop: "3pt double",
		borderRight: "3pt none",
		borderBottom: "3pt none",
		borderLeft: "3pt none"
	}, commonStyles)],
	["_", __assign({
		borderTop: "1px solid",
		borderRight: "1px none",
		borderBottom: "1px solid",
		borderLeft: "1px none"
	}, commonStyles)],
	["*", __assign({
		borderTop: "1px none",
		borderRight: "1px none",
		borderBottom: "3px dotted",
		borderLeft: "1px none"
	}, commonStyles)],
	["~", __assign({
		borderTop: "1px none",
		borderRight: "1px none",
		borderBottom: "1px solid",
		borderLeft: "1px none"
	}, commonStyles)],
	["#", __assign({
		borderTop: "3pt double",
		borderRight: "3pt none",
		borderBottom: "3pt double",
		borderLeft: "3pt none"
	}, commonStyles)]
]);
function insertHorizontalLineIntoModel(model, context, triggerChar) {
	var hr = createDivider("hr", HorizontalLineStyles.get(triggerChar));
	var doc = createContentModelDocument();
	addBlock(doc, hr);
	mergeModel(model, doc, context);
}
var checkAndInsertHorizontalLine = function(model, paragraph, context) {
	var blocks = getOperationalBlocks(model, ["ListItem"], ["TableCell", "FormatContainer"]);
	if (blocks[0] && blocks[0].block.blockType == "BlockGroup" && blocks[0].block.blockGroupType == "ListItem") return false;
	var allText = paragraph.segments.reduce(function(acc, segment) {
		return segment.segmentType === "Text" ? acc + segment.text : acc;
	}, "");
	if (allText.length < 3) return false;
	return HorizontalLineTriggerCharacters.some(function(triggerCharacter) {
		var shouldFormat = allText.split("").every(function(char) {
			return char === triggerCharacter;
		});
		if (shouldFormat) {
			paragraph.segments = paragraph.segments.filter(function(s$1) {
				return s$1.segmentType != "Text";
			});
			insertHorizontalLineIntoModel(model, context, triggerCharacter);
			context.canUndoByBackspace = true;
		}
		return shouldFormat;
	});
};
function createLink(editor, autoLinkOptions) {
	var anchorNode = null;
	var links = [];
	formatTextSegmentBeforeSelectionMarker(editor, function(_model, segment, paragraph) {
		var promotedSegment = null;
		if (segment.link) return false;
		else if ((promotedSegment = promoteLink(segment, paragraph, autoLinkOptions)) && promotedSegment.link) {
			links.push(promotedSegment.link);
			return true;
		} else return false;
	}, {
		changeSource: ChangeSource.AutoLink,
		onNodeCreated: function(modelElement, node) {
			if (!anchorNode && links.indexOf(modelElement) >= 0) anchorNode = node;
		},
		getChangeData: function() {
			return anchorNode;
		}
	});
}
function convertAlphaToDecimals(letter) {
	var alpha = letter.toUpperCase();
	if (alpha) {
		var result = 0;
		for (var i$1 = 0; i$1 < alpha.length; i$1++) {
			var charCode = alpha.charCodeAt(i$1) - 65 + 1;
			result = result * 26 + charCode;
		}
		return result;
	}
}
function getIndex$1(listIndex) {
	var index$1 = listIndex.replace(/[^a-zA-Z0-9 ]/g, "");
	var indexNumber = parseInt(index$1);
	return !isNaN(indexNumber) ? indexNumber : convertAlphaToDecimals(index$1);
}
var _a, _b, _c, _d, _e, _f;
var NumberingTypes;
(function(NumberingTypes$1) {
	NumberingTypes$1[NumberingTypes$1["Decimal"] = 1] = "Decimal";
	NumberingTypes$1[NumberingTypes$1["LowerAlpha"] = 2] = "LowerAlpha";
	NumberingTypes$1[NumberingTypes$1["UpperAlpha"] = 3] = "UpperAlpha";
	NumberingTypes$1[NumberingTypes$1["LowerRoman"] = 4] = "LowerRoman";
	NumberingTypes$1[NumberingTypes$1["UpperRoman"] = 5] = "UpperRoman";
})(NumberingTypes || (NumberingTypes = {}));
var Character;
(function(Character$1) {
	Character$1[Character$1["Dot"] = 1] = "Dot";
	Character$1[Character$1["Dash"] = 2] = "Dash";
	Character$1[Character$1["Parenthesis"] = 3] = "Parenthesis";
	Character$1[Character$1["DoubleParenthesis"] = 4] = "DoubleParenthesis";
})(Character || (Character = {}));
var characters = {
	".": 1,
	"-": 2,
	")": 3
};
var lowerRomanTypes = [
	NumberingListType.LowerRoman,
	NumberingListType.LowerRomanDash,
	NumberingListType.LowerRomanDoubleParenthesis,
	NumberingListType.LowerRomanParenthesis
];
var upperRomanTypes = [
	NumberingListType.UpperRoman,
	NumberingListType.UpperRomanDash,
	NumberingListType.UpperRomanDoubleParenthesis,
	NumberingListType.UpperRomanParenthesis
];
var numberingTriggers = [
	"1",
	"a",
	"A",
	"I",
	"i"
];
var lowerRomanNumbers = [
	"i",
	"v",
	"x",
	"l",
	"c",
	"d",
	"m"
];
var upperRomanNumbers = [
	"I",
	"V",
	"X",
	"L",
	"C",
	"D",
	"M"
];
var identifyNumberingType = function(text, previousListStyle) {
	if (!isNaN(parseInt(text))) return 1;
	else if (/[a-z]+/g.test(text)) {
		if (previousListStyle === 4 && lowerRomanTypes.indexOf(previousListStyle) > -1 && lowerRomanNumbers.indexOf(text[0]) > -1 || !previousListStyle && text === "i") return 4;
		else if (previousListStyle === 2 || !previousListStyle && text === "a") return 2;
	} else if (/[A-Z]+/g.test(text)) {
		if (previousListStyle == 5 && upperRomanTypes.indexOf(previousListStyle) > -1 && upperRomanNumbers.indexOf(text[0]) > -1 || !previousListStyle && text === "I") return 5;
		else if (previousListStyle == 3 || !previousListStyle && text === "A") return 3;
	}
};
var numberingListTypes = (_a = {}, _a[1] = function(char) {
	return DecimalsTypes[char] || void 0;
}, _a[2] = function(char) {
	return LowerAlphaTypes[char] || void 0;
}, _a[3] = function(char) {
	return UpperAlphaTypes[char] || void 0;
}, _a[4] = function(char) {
	return LowerRomanTypes[char] || void 0;
}, _a[5] = function(char) {
	return UpperRomanTypes[char] || void 0;
}, _a);
var UpperRomanTypes = (_b = {}, _b[1] = NumberingListType.UpperRoman, _b[2] = NumberingListType.UpperRomanDash, _b[3] = NumberingListType.UpperRomanParenthesis, _b[4] = NumberingListType.UpperRomanDoubleParenthesis, _b);
var LowerRomanTypes = (_c = {}, _c[1] = NumberingListType.LowerRoman, _c[2] = NumberingListType.LowerRomanDash, _c[3] = NumberingListType.LowerRomanParenthesis, _c[4] = NumberingListType.LowerRomanDoubleParenthesis, _c);
var UpperAlphaTypes = (_d = {}, _d[1] = NumberingListType.UpperAlpha, _d[2] = NumberingListType.UpperAlphaDash, _d[3] = NumberingListType.UpperAlphaParenthesis, _d[4] = NumberingListType.UpperAlphaDoubleParenthesis, _d);
var LowerAlphaTypes = (_e = {}, _e[1] = NumberingListType.LowerAlpha, _e[2] = NumberingListType.LowerAlphaDash, _e[3] = NumberingListType.LowerAlphaParenthesis, _e[4] = NumberingListType.LowerAlphaDoubleParenthesis, _e);
var DecimalsTypes = (_f = {}, _f[1] = NumberingListType.Decimal, _f[2] = NumberingListType.DecimalDash, _f[3] = NumberingListType.DecimalParenthesis, _f[4] = NumberingListType.DecimalDoubleParenthesis, _f);
var identifyNumberingListType = function(numbering, isDoubleParenthesis, previousListStyle) {
	var separatorCharacter = isDoubleParenthesis ? 4 : characters[numbering[numbering.length - 1]];
	if (separatorCharacter) {
		var numberingType = identifyNumberingType(isDoubleParenthesis ? numbering.slice(1, -1) : numbering.slice(0, -1), previousListStyle);
		return numberingType ? numberingListTypes[numberingType](separatorCharacter) : void 0;
	}
};
function getNumberingListStyle(textBeforeCursor, previousListIndex, previousListStyle) {
	var trigger = textBeforeCursor.trim();
	var isDoubleParenthesis = trigger[0] === "(" && trigger[trigger.length - 1] === ")";
	var listIndex = isDoubleParenthesis ? trigger.slice(1, -1) : trigger.slice(0, -1);
	var index$1 = getIndex$1(listIndex);
	var isContinuosList = numberingTriggers.indexOf(listIndex) < 0;
	if (!index$1 || index$1 < 1 || !previousListIndex && isContinuosList || previousListIndex && isContinuosList && !canAppendList(index$1, previousListIndex)) return;
	return isValidNumbering(listIndex) ? identifyNumberingListType(trigger, isDoubleParenthesis, isContinuosList ? previousListStyle : void 0) : void 0;
}
function isValidNumbering(index$1) {
	return Number(index$1) || /^[A-Za-z\s]*$/.test(index$1);
}
function canAppendList(index$1, previousListIndex) {
	return previousListIndex && index$1 && previousListIndex + 1 === index$1;
}
function getListTypeStyle(model, shouldSearchForBullet, shouldSearchForNumbering) {
	if (shouldSearchForBullet === void 0) shouldSearchForBullet = true;
	if (shouldSearchForNumbering === void 0) shouldSearchForNumbering = true;
	var selectedSegmentsAndParagraphs = getSelectedSegmentsAndParagraphs(model, true);
	if (!selectedSegmentsAndParagraphs[0]) return;
	var marker = selectedSegmentsAndParagraphs[0][0];
	var paragraph = selectedSegmentsAndParagraphs[0][1];
	var listMarkerSegment = paragraph === null || paragraph === void 0 ? void 0 : paragraph.segments[0];
	if (marker && marker.segmentType == "SelectionMarker" && listMarkerSegment && listMarkerSegment.segmentType == "Text") {
		var listMarker = listMarkerSegment.text.trim();
		var bulletType = bulletListType.get(listMarker);
		if (bulletType && shouldSearchForBullet) return {
			listType: "UL",
			styleType: bulletType
		};
		else if (shouldSearchForNumbering) {
			var _a$5 = getPreviousListLevel(model, paragraph), previousList = _a$5.previousList, hasSpaceBetween = _a$5.hasSpaceBetween;
			var previousIndex = getPreviousListIndex(model, previousList);
			var previousListStyle = getPreviousListStyle(previousList);
			var numberingType = getNumberingListStyle(listMarker, previousIndex, previousListStyle);
			if (numberingType) return {
				listType: "OL",
				styleType: numberingType,
				index: getIndex(listMarker, previousListStyle, numberingType, previousIndex, hasSpaceBetween)
			};
		}
	}
}
var getIndex = function(listMarker, previousListStyle, numberingType, previousIndex, hasSpaceBetween) {
	var newList = isNewList(listMarker);
	return previousListStyle && previousListStyle !== numberingType && newList ? 1 : !newList && previousListStyle === numberingType && hasSpaceBetween && previousIndex ? previousIndex + 1 : void 0;
};
var getPreviousListIndex = function(model, previousListItem) {
	return previousListItem ? findListItemsInSameThread(model, previousListItem).length : void 0;
};
var getPreviousListLevel = function(model, paragraph) {
	var blocks = getOperationalBlocks(model, ["ListItem"], ["TableCell"])[0];
	var previousList = void 0;
	var hasSpaceBetween = false;
	if (blocks) {
		var listBlockIndex = blocks.parent.blocks.indexOf(paragraph);
		if (listBlockIndex > -1) for (var i$1 = listBlockIndex - 1; i$1 > -1; i$1--) {
			var item = blocks.parent.blocks[i$1];
			if (isBlockGroupOfType(item, "ListItem")) {
				previousList = item;
				break;
			} else hasSpaceBetween = listBlockIndex > 0 ? true : false;
		}
	}
	return {
		previousList,
		hasSpaceBetween
	};
};
var getPreviousListStyle = function(list) {
	var _a$5;
	if (!list || list.levels.length < 1) return;
	return (_a$5 = updateListMetadata(list.levels[0])) === null || _a$5 === void 0 ? void 0 : _a$5.orderedStyleType;
};
var bulletListType = new Map([
	["*", BulletListType.Disc],
	["-", BulletListType.Dash],
	["--", BulletListType.Square],
	["->", BulletListType.LongArrow],
	["-->", BulletListType.DoubleLongArrow],
	["=>", BulletListType.UnfilledArrow],
	[">", BulletListType.ShortArrow],
	["—", BulletListType.Hyphen]
]);
var isNewList = function(listMarker) {
	var marker = listMarker.replace(/[^\w\s]/g, "");
	return /^[1aAiI]$/.test(marker);
};
function keyboardListTrigger(model, paragraph, context, shouldSearchForBullet, shouldSearchForNumbering, removeListMargins) {
	if (shouldSearchForBullet === void 0) shouldSearchForBullet = true;
	if (shouldSearchForNumbering === void 0) shouldSearchForNumbering = true;
	var listStyleType = getListTypeStyle(model, shouldSearchForBullet, shouldSearchForNumbering);
	if (listStyleType) {
		paragraph.segments.splice(0, 1);
		var listType = listStyleType.listType, styleType = listStyleType.styleType, index$1 = listStyleType.index;
		triggerList(model, listType, styleType, index$1, removeListMargins);
		context.canUndoByBackspace = true;
		setAnnounceData(model, context);
		return true;
	}
	return false;
}
var triggerList = function(model, listType, styleType, index$1, removeListMargins) {
	setListType(model, listType, removeListMargins);
	var isOrderedList = listType == "OL";
	if (index$1 && index$1 > 0 && isOrderedList) setModelListStartNumber(model, index$1);
	setModelListStyle(model, isOrderedList ? {
		orderedStyleType: styleType,
		applyListStyleFromLevel: false
	} : {
		unorderedStyleType: styleType,
		applyListStyleFromLevel: false
	});
};
function setAnnounceData(model, context) {
	var paragraphOrListItems = __read(getOperationalBlocks(model, ["ListItem"], []), 1)[0];
	if (paragraphOrListItems && isBlockGroupOfType(paragraphOrListItems.block, "ListItem")) {
		var path = paragraphOrListItems.path, block = paragraphOrListItems.block;
		context.announceData = getListAnnounceData(__spreadArray([block], __read(path), false));
	}
}
var FRACTIONS = new Map([
	["1/2", "½"],
	["1/4", "¼"],
	["3/4", "¾"]
]);
function transformFraction(previousSegment, paragraph, context) {
	var _a$5;
	var fraction = (_a$5 = previousSegment.text.split(" ").pop()) === null || _a$5 === void 0 ? void 0 : _a$5.trim();
	var text = fraction ? FRACTIONS.get(fraction) : void 0;
	if (fraction && text) {
		var textLength = previousSegment.text.length - 1;
		var textSegment = splitTextSegment(previousSegment, paragraph, textLength - fraction.length, textLength);
		textSegment.text = text;
		context.canUndoByBackspace = true;
		return true;
	}
	return false;
}
function transformHyphen(previousSegment, paragraph, context) {
	var segments = previousSegment.text.split(" ");
	if (segments[segments.length - 2] === "--") {
		var textIndex = previousSegment.text.lastIndexOf("--");
		var textSegment = splitTextSegment(previousSegment, paragraph, textIndex, textIndex + 2);
		textSegment.text = textSegment.text.replace("--", "—");
		context.canUndoByBackspace = true;
		return true;
	} else {
		var text = segments.pop();
		if (text && (text === null || text === void 0 ? void 0 : text.indexOf("--")) > -1 && text.trim() !== "--") {
			var textIndex = previousSegment.text.indexOf(text);
			var textSegment = splitTextSegment(previousSegment, paragraph, textIndex, textIndex + text.length - 1);
			var textLength = textSegment.text.length;
			if (textSegment.text[0] !== "-" && textSegment.text[textLength - 1] !== "-") {
				textSegment.text = textSegment.text.replace("--", "—");
				context.canUndoByBackspace = true;
				return true;
			}
		}
	}
	return false;
}
var getOrdinal = function(value) {
	return {
		1: "st",
		2: "nd",
		3: "rd"
	}[value] || "th";
};
var ORDINALS = [
	"st",
	"nd",
	"rd",
	"th"
];
var ORDINAL_LENGTH = 2;
function transformOrdinals(previousSegment, paragraph, context) {
	var _a$5;
	var value = (_a$5 = previousSegment.text.split(" ").pop()) === null || _a$5 === void 0 ? void 0 : _a$5.trim();
	var shouldAddSuperScript = false;
	if (value) {
		if (ORDINALS.indexOf(value) > -1) {
			var index$1 = paragraph.segments.indexOf(previousSegment);
			var numberSegment = paragraph.segments[index$1 - 1];
			var numericValue = null;
			if (numberSegment && numberSegment.segmentType == "Text" && (numericValue = getNumericValue(numberSegment.text, true)) && getOrdinal(numericValue) === value) shouldAddSuperScript = true;
		} else {
			var ordinal = value.substring(value.length - ORDINAL_LENGTH);
			var numericValue = getNumericValue(value);
			if (numericValue && getOrdinal(numericValue) === ordinal) shouldAddSuperScript = true;
		}
		if (shouldAddSuperScript) {
			var ordinalSegment = splitTextSegment(previousSegment, paragraph, previousSegment.text.length - 3, previousSegment.text.length - 1);
			ordinalSegment.format.superOrSubScriptSequence = "super";
			context.canUndoByBackspace = true;
		}
	}
	return shouldAddSuperScript;
}
function getNumericValue(text, checkFullText) {
	if (checkFullText === void 0) checkFullText = false;
	var number = checkFullText ? text : text.substring(0, text.length - ORDINAL_LENGTH);
	if (/^-?\d+$/.test(number)) {
		var numericValue = parseInt(number);
		return Math.abs(numericValue) < 20 ? numericValue : parseInt(number.substring(number.length - 1));
	}
	return null;
}
function unlink(editor, rawEvent) {
	formatTextSegmentBeforeSelectionMarker(editor, function(_model, linkSegment, _paragraph) {
		if (linkSegment === null || linkSegment === void 0 ? void 0 : linkSegment.link) {
			linkSegment.link = void 0;
			rawEvent.preventDefault();
			return true;
		}
		return false;
	});
}
var DefaultOptions$2 = {
	autoBullet: false,
	autoNumbering: false,
	autoUnlink: false,
	autoLink: false,
	autoHyphen: false,
	autoFraction: false,
	autoOrdinals: false,
	removeListMargins: false,
	autoHorizontalLine: false
};
(function() {
	function AutoFormatPlugin$1(options) {
		var _this = this;
		if (options === void 0) options = DefaultOptions$2;
		this.options = options;
		this.editor = null;
		this.autoLink = {
			enabled: !!(this.options.autoLink || this.options.autoTel || this.options.autoMailto),
			transformFunction: function(_model, previousSegment, paragraph, context) {
				var _a$5;
				var _b$1 = _this.options, autoLink = _b$1.autoLink, autoTel = _b$1.autoTel, autoMailto = _b$1.autoMailto;
				var linkSegment = promoteLink(previousSegment, paragraph, {
					autoLink,
					autoTel,
					autoMailto
				});
				if (linkSegment) return createAnchor(((_a$5 = linkSegment.link) === null || _a$5 === void 0 ? void 0 : _a$5.format.href) || "", linkSegment.text);
				return false;
			},
			apiName: "autoLink",
			changeSource: ChangeSource.AutoLink
		};
		this.tabFeatures = [{
			enabled: !!(this.options.autoBullet || this.options.autoNumbering),
			transformFunction: function(model, _previousSegment, paragraph, context) {
				return keyboardListTrigger(model, paragraph, context, _this.options.autoBullet, _this.options.autoNumbering, _this.options.removeListMargins);
			},
			apiName: "autoToggleList",
			changeSource: ChangeSource.AutoFormat
		}, this.autoLink];
		this.features = __spreadArray(__spreadArray([], __read(this.tabFeatures), false), [
			{
				enabled: !!this.options.autoHyphen,
				apiName: "autoHyphen",
				changeSource: ChangeSource.Format,
				transformFunction: function(_model, previousSegment, paragraph, context) {
					return transformHyphen(previousSegment, paragraph, context);
				}
			},
			{
				enabled: !!this.options.autoFraction,
				apiName: "autoFraction",
				changeSource: ChangeSource.Format,
				transformFunction: function(_model, previousSegment, paragraph, context) {
					return transformFraction(previousSegment, paragraph, context);
				}
			},
			{
				enabled: !!this.options.autoOrdinals,
				apiName: "autoOrdinal",
				changeSource: ChangeSource.Format,
				transformFunction: function(_model, previousSegment, paragraph, context) {
					return transformOrdinals(previousSegment, paragraph, context);
				}
			}
		], false);
		this.enterFeatures = [{
			enabled: !!this.options.autoHorizontalLine,
			transformFunction: function(model, _previousSegment, paragraph, context) {
				return checkAndInsertHorizontalLine(model, paragraph, context);
			},
			apiName: "autoHorizontalLine",
			changeSource: ChangeSource.AutoFormat
		}, this.autoLink];
	}
	AutoFormatPlugin$1.prototype.getName = function() {
		return "AutoFormat";
	};
	AutoFormatPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
	};
	AutoFormatPlugin$1.prototype.dispose = function() {
		this.editor = null;
	};
	AutoFormatPlugin$1.prototype.shouldHandleInputEventExclusively = function(editor, event$1) {
		var _this = this;
		var rawEvent = event$1.rawEvent;
		var selection = editor.getDOMSelection();
		var shouldHandle = false;
		if (rawEvent.inputType === "insertText" && selection && selection.type === "range" && selection.range.collapsed && rawEvent.data == " ") formatTextSegmentBeforeSelectionMarker(editor, function(model, previousSegment, paragraph) {
			var _a$5 = _this.options, autoLink = _a$5.autoLink, autoTel = _a$5.autoTel, autoMailto = _a$5.autoMailto, autoBullet = _a$5.autoBullet, autoNumbering = _a$5.autoNumbering;
			var list = getListTypeStyle(model, autoBullet, autoNumbering);
			shouldHandle = !!promoteLink(previousSegment, paragraph, {
				autoLink,
				autoTel,
				autoMailto
			}) || !!list;
			return false;
		});
		return shouldHandle;
	};
	AutoFormatPlugin$1.prototype.willHandleEventExclusively = function(event$1) {
		if (this.editor) switch (event$1.eventType) {
			case "input": return this.shouldHandleInputEventExclusively(this.editor, event$1);
		}
		return false;
	};
	AutoFormatPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (this.editor) switch (event$1.eventType) {
			case "input":
				this.handleEditorInputEvent(this.editor, event$1);
				break;
			case "keyDown":
				this.handleKeyDownEvent(this.editor, event$1);
				break;
			case "contentChanged":
				this.handleContentChangedEvent(this.editor, event$1);
				break;
		}
	};
	AutoFormatPlugin$1.prototype.handleKeyboardEvents = function(editor, features) {
		var formatOptions = {
			changeSource: "",
			apiName: "",
			getChangeData: void 0
		};
		formatTextSegmentBeforeSelectionMarker(editor, function(model, previousSegment, paragraph, _markerFormat, context) {
			var e_1, _a$5;
			var featureApplied = void 0;
			var _loop_1 = function(feature$1) {
				if (feature$1.enabled) {
					var result_1 = feature$1.transformFunction(model, previousSegment, paragraph, context);
					if (result_1) {
						if (typeof result_1 !== "boolean") formatOptions.getChangeData = function() {
							return result_1;
						};
						featureApplied = feature$1;
						return "break";
					}
				}
			};
			try {
				for (var features_1 = __values(features), features_1_1 = features_1.next(); !features_1_1.done; features_1_1 = features_1.next()) {
					var feature = features_1_1.value;
					if (_loop_1(feature) === "break") break;
				}
			} catch (e_1_1) {
				e_1 = { error: e_1_1 };
			} finally {
				try {
					if (features_1_1 && !features_1_1.done && (_a$5 = features_1.return)) _a$5.call(features_1);
				} finally {
					if (e_1) throw e_1.error;
				}
			}
			if (featureApplied) {
				formatOptions.changeSource = featureApplied.changeSource;
				formatOptions.apiName = featureApplied.apiName;
			}
			return !!featureApplied;
		}, formatOptions);
		return formatOptions;
	};
	AutoFormatPlugin$1.prototype.handleEditorInputEvent = function(editor, event$1) {
		var rawEvent = event$1.rawEvent;
		var selection = editor.getDOMSelection();
		if (rawEvent.inputType === "insertText" && selection && selection.type === "range" && selection.range.collapsed) switch (rawEvent.data) {
			case " ":
				this.handleKeyboardEvents(editor, this.features);
				break;
		}
	};
	AutoFormatPlugin$1.prototype.handleKeyDownEvent = function(editor, event$1) {
		var rawEvent = event$1.rawEvent;
		if (!rawEvent.defaultPrevented && !event$1.handledByEditFeature) switch (rawEvent.key) {
			case "Backspace":
				if (this.options.autoUnlink) unlink(editor, rawEvent);
				break;
			case "Tab":
				if (!rawEvent.shiftKey) {
					if (this.handleKeyboardEvents(editor, this.tabFeatures).apiName == "autoToggleList") event$1.rawEvent.preventDefault();
				}
				break;
			case "Enter":
				if (this.handleKeyboardEvents(editor, this.enterFeatures).apiName == "autoHorizontalLine") event$1.rawEvent.preventDefault();
				break;
		}
	};
	AutoFormatPlugin$1.prototype.handleContentChangedEvent = function(editor, event$1) {
		var _a$5 = this.options, autoLink = _a$5.autoLink, autoTel = _a$5.autoTel, autoMailto = _a$5.autoMailto;
		if (event$1.source == "Paste" && (autoLink || autoTel || autoMailto)) createLink(editor, {
			autoLink,
			autoTel,
			autoMailto
		});
	};
	return AutoFormatPlugin$1;
})();
var createAnchor = function(url, text) {
	var anchor = document.createElement("a");
	anchor.href = url;
	anchor.textContent = text;
	return anchor;
};
function setShortcutIndentationCommand(editor, operation) {
	editor.formatContentModel(function(model, context) {
		var listItem = getFirstSelectedListItem(model);
		if (listItem && listItem.blocks[0].blockType == "Paragraph" && listItem.blocks[0].segments[0].segmentType == "SelectionMarker") {
			setModelIndentation(model, operation, void 0, context);
			return true;
		}
		return false;
	});
}
var Keys;
(function(Keys$1) {
	Keys$1[Keys$1["BACKSPACE"] = 8] = "BACKSPACE";
	Keys$1[Keys$1["SPACE"] = 32] = "SPACE";
	Keys$1[Keys$1["B"] = 66] = "B";
	Keys$1[Keys$1["I"] = 73] = "I";
	Keys$1[Keys$1["U"] = 85] = "U";
	Keys$1[Keys$1["Y"] = 89] = "Y";
	Keys$1[Keys$1["Z"] = 90] = "Z";
	Keys$1[Keys$1["COMMA"] = 188] = "COMMA";
	Keys$1[Keys$1["PERIOD"] = 190] = "PERIOD";
	Keys$1[Keys$1["FORWARD_SLASH"] = 191] = "FORWARD_SLASH";
	Keys$1[Keys$1["ArrowRight"] = 39] = "ArrowRight";
	Keys$1[Keys$1["ArrowLeft"] = 37] = "ArrowLeft";
})(Keys || (Keys = {}));
var defaultShortcuts = [
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: false,
			which: 66
		},
		onClick: function(editor) {
			return toggleBold(editor, { announceFormatChange: true });
		}
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: false,
			which: 73
		},
		onClick: function(editor) {
			return toggleItalic(editor, { announceFormatChange: true });
		}
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: false,
			which: 85
		},
		onClick: function(editor) {
			return toggleUnderline(editor, { announceFormatChange: true });
		}
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: false,
			which: 32
		},
		onClick: function(editor) {
			return clearFormat(editor);
		}
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: false,
			which: 90
		},
		onClick: function(editor) {
			return undo(editor);
		}
	},
	{
		shortcutKey: {
			modifierKey: "alt",
			shiftKey: false,
			which: 8
		},
		onClick: function(editor) {
			return undo(editor);
		},
		environment: "nonMac"
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: false,
			which: 89
		},
		onClick: function(editor) {
			return redo(editor);
		},
		environment: "nonMac"
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: true,
			which: 90
		},
		onClick: function(editor) {
			return redo(editor);
		}
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: true,
			which: 90
		},
		onClick: function(editor) {
			return redo(editor);
		},
		environment: "mac"
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: false,
			which: 190
		},
		onClick: function(editor) {
			return toggleBullet(editor);
		}
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: false,
			which: 191
		},
		onClick: function(editor) {
			return toggleNumbering(editor);
		}
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: true,
			which: 190
		},
		onClick: function(editor) {
			return changeFontSize(editor, "increase");
		}
	},
	{
		shortcutKey: {
			modifierKey: "ctrl",
			shiftKey: true,
			which: 188
		},
		onClick: function(editor) {
			return changeFontSize(editor, "decrease");
		}
	},
	{
		shortcutKey: {
			modifierKey: "alt",
			shiftKey: true,
			which: 39
		},
		onClick: function(editor) {
			setShortcutIndentationCommand(editor, "indent");
		},
		environment: "nonMac"
	},
	{
		shortcutKey: {
			modifierKey: "alt",
			shiftKey: true,
			which: 37
		},
		onClick: function(editor) {
			setShortcutIndentationCommand(editor, "outdent");
		},
		environment: "nonMac"
	}
];
var CommandCacheKey = "__ShortcutCommandCache";
var ShortcutPlugin = function() {
	function ShortcutPlugin$1(shortcuts) {
		if (shortcuts === void 0) shortcuts = defaultShortcuts;
		this.shortcuts = shortcuts;
		this.editor = null;
		this.isMac = false;
	}
	ShortcutPlugin$1.prototype.getName = function() {
		return "Shortcut";
	};
	ShortcutPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		this.isMac = !!this.editor.getEnvironment().isMac;
	};
	ShortcutPlugin$1.prototype.dispose = function() {
		this.editor = null;
	};
	ShortcutPlugin$1.prototype.willHandleEventExclusively = function(event$1) {
		return event$1.eventType == "keyDown" && (event$1.rawEvent.ctrlKey || event$1.rawEvent.altKey || event$1.rawEvent.metaKey) && !!this.cacheGetCommand(event$1);
	};
	ShortcutPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (this.editor && event$1.eventType == "keyDown") {
			var command = this.cacheGetCommand(event$1);
			if (command) {
				command.onClick(this.editor);
				event$1.rawEvent.preventDefault();
			}
		}
	};
	ShortcutPlugin$1.prototype.cacheGetCommand = function(event$1) {
		var _this = this;
		return cacheGetEventData(event$1, CommandCacheKey, function(event$2) {
			var editor = _this.editor;
			var _a$5 = event$2.rawEvent, ctrlKey = _a$5.ctrlKey, metaKey = _a$5.metaKey;
			if (ctrlKey && metaKey) return null;
			return editor && _this.shortcuts.filter(function(command) {
				return _this.matchOS(command.environment) && _this.matchShortcut(command.shortcutKey, event$2.rawEvent);
			})[0];
		});
	};
	ShortcutPlugin$1.prototype.matchOS = function(environment) {
		switch (environment) {
			case "mac": return this.isMac;
			case "nonMac": return !this.isMac;
			default: return true;
		}
	};
	ShortcutPlugin$1.prototype.matchShortcut = function(shortcutKey, event$1) {
		var ctrlKey = event$1.ctrlKey, altKey = event$1.altKey, shiftKey = event$1.shiftKey, which = event$1.which, metaKey = event$1.metaKey;
		var ctrlOrMeta = this.isMac ? metaKey : ctrlKey;
		return (shortcutKey.modifierKey == "ctrl" && ctrlOrMeta && !altKey || shortcutKey.modifierKey == "alt" && altKey && !ctrlOrMeta) && shiftKey == shortcutKey.shiftKey && shortcutKey.which == which;
	};
	return ShortcutPlugin$1;
}();
(function() {
	function ContextMenuPluginBase$1(options) {
		var _this = this;
		this.options = options;
		this.container = null;
		this.editor = null;
		this.isMenuShowing = false;
		this.onDismiss = function() {
			var _a$5, _b$1;
			if (_this.container && _this.isMenuShowing) {
				(_b$1 = (_a$5 = _this.options).dismiss) === null || _b$1 === void 0 || _b$1.call(_a$5, _this.container);
				_this.isMenuShowing = false;
			}
		};
	}
	ContextMenuPluginBase$1.prototype.getName = function() {
		return "ContextMenu";
	};
	ContextMenuPluginBase$1.prototype.initialize = function(editor) {
		this.editor = editor;
	};
	ContextMenuPluginBase$1.prototype.dispose = function() {
		var _a$5;
		this.onDismiss();
		if ((_a$5 = this.container) === null || _a$5 === void 0 ? void 0 : _a$5.parentNode) {
			this.container.parentNode.removeChild(this.container);
			this.container = null;
		}
		this.editor = null;
	};
	ContextMenuPluginBase$1.prototype.onPluginEvent = function(event$1) {
		if (event$1.eventType == "contextMenu" && event$1.items.length > 0) {
			var rawEvent = event$1.rawEvent, items = event$1.items;
			this.onDismiss();
			if (!this.options.allowDefaultMenu) rawEvent.preventDefault();
			if (this.initContainer(rawEvent.pageX, rawEvent.pageY)) {
				this.options.render(this.container, items, this.onDismiss);
				this.isMenuShowing = true;
			}
		}
	};
	ContextMenuPluginBase$1.prototype.initContainer = function(x$1, y$1) {
		var _a$5, _b$1;
		if (!this.container && this.editor) {
			this.container = this.editor.getDocument().createElement("div");
			this.container.style.position = "fixed";
			this.container.style.width = "0";
			this.container.style.height = "0";
			this.editor.getDocument().body.appendChild(this.container);
		}
		(_a$5 = this.container) === null || _a$5 === void 0 || _a$5.style.setProperty("left", x$1 + "px");
		(_b$1 = this.container) === null || _b$1 === void 0 || _b$1.style.setProperty("top", y$1 + "px");
		return !!this.container;
	};
	return ContextMenuPluginBase$1;
})();
function isModelEmptyFast(model) {
	var firstBlock = model.blocks[0];
	if (model.blocks.length > 1) return false;
	else if (!firstBlock) return true;
	else if (firstBlock.blockType != "Paragraph") return false;
	else if (firstBlock.segments.length == 0) return true;
	else if (firstBlock.segments.some(function(x$1) {
		return x$1.segmentType == "Entity" || x$1.segmentType == "Image" || x$1.segmentType == "General" || x$1.segmentType == "Text" && x$1.text;
	})) return false;
	else if (firstBlock.format.marginRight && parseFloat(firstBlock.format.marginRight) > 0 || firstBlock.format.marginLeft && parseFloat(firstBlock.format.marginLeft) > 0) return false;
	else return firstBlock.segments.filter(function(x$1) {
		return x$1.segmentType == "Br";
	}).length <= 1;
}
var WATERMARK_CONTENT_KEY = "_WatermarkContent";
var styleMap = {
	fontFamily: "font-family",
	fontSize: "font-size",
	textColor: "color"
};
(function() {
	function WatermarkPlugin$1(watermark, format) {
		var _this = this;
		this.watermark = watermark;
		this.editor = null;
		this.isShowing = false;
		this.darkTextColor = null;
		this.disposer = null;
		this.onCompositionStart = function() {
			if (_this.editor) _this.showHide(_this.editor, false);
		};
		this.format = format || {
			fontSize: "14px",
			textColor: "#AAAAAA"
		};
	}
	WatermarkPlugin$1.prototype.getName = function() {
		return "Watermark";
	};
	WatermarkPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		this.disposer = this.editor.attachDomEvent({ compositionstart: { beforeDispatch: this.onCompositionStart } });
	};
	WatermarkPlugin$1.prototype.dispose = function() {
		var _a$5;
		(_a$5 = this.disposer) === null || _a$5 === void 0 || _a$5.call(this);
		this.disposer = null;
		this.editor = null;
	};
	WatermarkPlugin$1.prototype.onPluginEvent = function(event$1) {
		var editor = this.editor;
		if (!editor) return;
		if (event$1.eventType == "input" && event$1.rawEvent.inputType == "insertText") this.showHide(editor, false);
		else if (event$1.eventType == "contentChanged" && (event$1.source == ChangeSource.SwitchToDarkMode || event$1.source == ChangeSource.SwitchToLightMode) && this.isShowing) {
			if (event$1.source == ChangeSource.SwitchToDarkMode && !this.darkTextColor && this.format.textColor) this.darkTextColor = editor.getColorManager().getDarkColor(this.format.textColor, void 0, "text");
			this.applyWatermarkStyle(editor);
		} else if (event$1.eventType == "editorReady" || event$1.eventType == "contentChanged" || event$1.eventType == "input" || event$1.eventType == "beforeDispose" || event$1.eventType == "compositionEnd") this.update(editor);
	};
	WatermarkPlugin$1.prototype.update = function(editor) {
		var _this = this;
		editor.formatContentModel(function(model) {
			var isEmpty$1 = isModelEmptyFast(model);
			_this.showHide(editor, isEmpty$1);
			return false;
		});
	};
	WatermarkPlugin$1.prototype.showHide = function(editor, isEmpty$1) {
		if (this.isShowing && !isEmpty$1) this.hide(editor);
		else if (!this.isShowing && isEmpty$1) this.show(editor);
	};
	WatermarkPlugin$1.prototype.show = function(editor) {
		this.applyWatermarkStyle(editor);
		this.isShowing = true;
	};
	WatermarkPlugin$1.prototype.applyWatermarkStyle = function(editor) {
		var rule = "position: absolute; pointer-events: none; margin-inline-start: 1px; content: \"" + this.watermark + "\";";
		var format = __assign(__assign({}, this.format), { textColor: editor.isDarkMode() ? this.darkTextColor : this.format.textColor });
		getObjectKeys(styleMap).forEach(function(x$1) {
			if (format[x$1]) rule += styleMap[x$1] + ": " + format[x$1] + "!important;";
		});
		editor.setEditorStyle(WATERMARK_CONTENT_KEY, rule, "before");
	};
	WatermarkPlugin$1.prototype.hide = function(editor) {
		editor.setEditorStyle(WATERMARK_CONTENT_KEY, null);
		this.isShowing = false;
	};
	return WatermarkPlugin$1;
})();
function setFormat(editor, character, format, codeFormat) {
	formatTextSegmentBeforeSelectionMarker(editor, function(_model, previousSegment, paragraph, markerFormat, context) {
		if (previousSegment.text[previousSegment.text.length - 1] == character) {
			var textSegment = previousSegment.text;
			var textBeforeMarker = textSegment.slice(0, -1);
			context.newPendingFormat = __assign(__assign({}, markerFormat), {
				strikethrough: !!markerFormat.strikethrough,
				italic: !!markerFormat.italic,
				fontWeight: (markerFormat === null || markerFormat === void 0 ? void 0 : markerFormat.fontWeight) ? "bold" : void 0
			});
			if (textBeforeMarker.indexOf(character) > -1) {
				var lastCharIndex = textSegment.length;
				var firstCharIndex = textSegment.substring(0, lastCharIndex - 1).lastIndexOf(character);
				if (hasSpaceBeforeFirstCharacter(textSegment, firstCharIndex) && lastCharIndex - firstCharIndex > 2) {
					var formattedText = splitTextSegment(previousSegment, paragraph, firstCharIndex, lastCharIndex);
					formattedText.text = formattedText.text.replace(character, "").slice(0, -1);
					formattedText.format = __assign(__assign({}, formattedText.format), format);
					if (codeFormat) formattedText.code = { format: codeFormat };
					context.canUndoByBackspace = true;
					return true;
				}
			}
		}
		return false;
	});
}
function hasSpaceBeforeFirstCharacter(text, index$1) {
	return !text[index$1 - 1] || text[index$1 - 1].trim().length == 0;
}
var DefaultOptions$1 = {
	strikethrough: false,
	bold: false,
	italic: false
};
(function() {
	function MarkdownPlugin$1(options) {
		if (options === void 0) options = DefaultOptions$1;
		this.options = options;
		this.editor = null;
		this.shouldBold = false;
		this.shouldItalic = false;
		this.shouldStrikethrough = false;
		this.shouldCode = false;
		this.lastKeyTyped = null;
	}
	MarkdownPlugin$1.prototype.getName = function() {
		return "Markdown";
	};
	MarkdownPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
	};
	MarkdownPlugin$1.prototype.dispose = function() {
		this.editor = null;
		this.disableAllFeatures();
		this.lastKeyTyped = null;
	};
	MarkdownPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (this.editor) switch (event$1.eventType) {
			case "input":
				this.handleEditorInputEvent(this.editor, event$1);
				break;
			case "keyDown":
				this.handleBackspaceEvent(event$1);
				this.handleKeyDownEvent(event$1);
				break;
			case "contentChanged":
				this.handleContentChangedEvent(event$1);
				break;
		}
	};
	MarkdownPlugin$1.prototype.handleEditorInputEvent = function(editor, event$1) {
		var rawEvent = event$1.rawEvent;
		var selection = editor.getDOMSelection();
		if (selection && selection.type == "range" && selection.range.collapsed && rawEvent.inputType == "insertText") switch (rawEvent.data) {
			case "*":
				if (this.options.bold) if (this.shouldBold) {
					setFormat(editor, "*", { fontWeight: "bold" });
					this.shouldBold = false;
				} else this.shouldBold = true;
				break;
			case "~":
				if (this.options.strikethrough) if (this.shouldStrikethrough) {
					setFormat(editor, "~", { strikethrough: true });
					this.shouldStrikethrough = false;
				} else this.shouldStrikethrough = true;
				break;
			case "_":
				if (this.options.italic) if (this.shouldItalic) {
					setFormat(editor, "_", { italic: true });
					this.shouldItalic = false;
				} else this.shouldItalic = true;
				break;
			case "`":
				if (this.options.codeFormat) if (this.shouldCode) {
					setFormat(editor, "`", {}, this.options.codeFormat);
					this.shouldCode = false;
				} else this.shouldCode = true;
				break;
		}
	};
	MarkdownPlugin$1.prototype.handleKeyDownEvent = function(event$1) {
		var rawEvent = event$1.rawEvent;
		if (!event$1.handledByEditFeature && !rawEvent.defaultPrevented) switch (rawEvent.key) {
			case "Enter":
				this.disableAllFeatures();
				this.lastKeyTyped = null;
				break;
			case " ":
				if (this.lastKeyTyped === "*" && this.shouldBold) this.shouldBold = false;
				else if (this.lastKeyTyped === "~" && this.shouldStrikethrough) this.shouldStrikethrough = false;
				else if (this.lastKeyTyped === "_" && this.shouldItalic) this.shouldItalic = false;
				else if (this.lastKeyTyped === "`" && this.shouldCode) this.shouldCode = false;
				this.lastKeyTyped = null;
				break;
			default:
				this.lastKeyTyped = rawEvent.key;
				break;
		}
	};
	MarkdownPlugin$1.prototype.handleBackspaceEvent = function(event$1) {
		if (!event$1.handledByEditFeature && event$1.rawEvent.key === "Backspace") {
			if (this.lastKeyTyped === "*" && this.shouldBold) this.shouldBold = false;
			else if (this.lastKeyTyped === "~" && this.shouldStrikethrough) this.shouldStrikethrough = false;
			else if (this.lastKeyTyped === "_" && this.shouldItalic) this.shouldItalic = false;
			else if (this.lastKeyTyped === "`" && this.shouldCode) this.shouldCode = false;
			this.lastKeyTyped = null;
		}
	};
	MarkdownPlugin$1.prototype.handleContentChangedEvent = function(event$1) {
		if (event$1.source == "Format") this.disableAllFeatures();
	};
	MarkdownPlugin$1.prototype.disableAllFeatures = function() {
		this.shouldBold = false;
		this.shouldItalic = false;
		this.shouldStrikethrough = false;
		this.shouldCode = false;
	};
	return MarkdownPlugin$1;
})();
var defaultToolTipCallback = function(url) {
	return url;
};
(function() {
	function HyperlinkPlugin$1(tooltip, target, onLinkClick) {
		var _this = this;
		if (tooltip === void 0) tooltip = defaultToolTipCallback;
		this.tooltip = tooltip;
		this.target = target;
		this.onLinkClick = onLinkClick;
		this.editor = null;
		this.domHelper = null;
		this.isMac = false;
		this.disposer = null;
		this.currentNode = null;
		this.currentLink = null;
		this.onMouse = function(e$1) {
			_this.runWithHyperlink(e$1.target, function(href, a$1) {
				var _a$5;
				var tooltip$1 = e$1.type == "mouseover" ? typeof _this.tooltip == "function" ? _this.tooltip(href, a$1) : _this.tooltip : null;
				(_a$5 = _this.domHelper) === null || _a$5 === void 0 || _a$5.setDomAttribute("title", tooltip$1);
			});
		};
	}
	HyperlinkPlugin$1.prototype.getName = function() {
		return "Hyperlink";
	};
	HyperlinkPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		this.domHelper = editor.getDOMHelper();
		this.isMac = !!editor.getEnvironment().isMac;
		this.disposer = editor.attachDomEvent({
			mouseover: { beforeDispatch: this.onMouse },
			mouseout: { beforeDispatch: this.onMouse }
		});
	};
	HyperlinkPlugin$1.prototype.dispose = function() {
		if (this.disposer) {
			this.disposer();
			this.disposer = null;
		}
		this.currentNode = null;
		this.currentLink = null;
		this.editor = null;
	};
	HyperlinkPlugin$1.prototype.onPluginEvent = function(event$1) {
		var _this = this;
		var _a$5, _b$1, _c$1;
		var matchedLink;
		if (event$1.eventType == "keyDown") {
			var selection = (_a$5 = this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getDOMSelection();
			var node_1 = (selection === null || selection === void 0 ? void 0 : selection.type) == "range" ? selection.range.commonAncestorContainer : null;
			if (node_1 && node_1 != this.currentNode) {
				this.currentNode = node_1;
				this.currentLink = null;
				this.runWithHyperlink(node_1, function(href, a$1) {
					if (node_1.textContent && (matchedLink = matchLink(node_1.textContent)) && matchedLink.normalizedUrl == href) _this.currentLink = a$1;
				});
			}
		} else if (event$1.eventType == "keyUp") {
			var selection = (_b$1 = this.editor) === null || _b$1 === void 0 ? void 0 : _b$1.getDOMSelection();
			var node = (selection === null || selection === void 0 ? void 0 : selection.type) == "range" ? selection.range.commonAncestorContainer : null;
			if (node && node == this.currentNode && this.currentLink && this.currentLink.contains(node) && node.textContent && (matchedLink = matchLink(node.textContent))) this.currentLink.setAttribute("href", matchedLink.normalizedUrl);
		} else if (event$1.eventType == "mouseUp" && event$1.isClicking) this.runWithHyperlink(event$1.rawEvent.target, function(href, anchor) {
			var _a$6, _b$2;
			if (!((_a$6 = _this.onLinkClick) === null || _a$6 === void 0 ? void 0 : _a$6.call(_this, anchor, event$1.rawEvent)) && _this.isCtrlOrMetaPressed(event$1.rawEvent) && event$1.rawEvent.button === 0) {
				event$1.rawEvent.preventDefault();
				var target = _this.target || "_blank";
				var window_1 = (_b$2 = _this.editor) === null || _b$2 === void 0 ? void 0 : _b$2.getDocument().defaultView;
				try {
					window_1 === null || window_1 === void 0 || window_1.open(href, target);
				} catch (_c$2) {}
			}
		});
		else if (event$1.eventType == "contentChanged") (_c$1 = this.domHelper) === null || _c$1 === void 0 || _c$1.setDomAttribute("title", null);
	};
	HyperlinkPlugin$1.prototype.runWithHyperlink = function(node, callback) {
		var _a$5;
		var a$1 = (_a$5 = this.domHelper) === null || _a$5 === void 0 ? void 0 : _a$5.findClosestElementAncestor(node, "a[href]");
		var href = a$1 === null || a$1 === void 0 ? void 0 : a$1.getAttribute("href");
		if (href && a$1) callback(href, a$1);
	};
	HyperlinkPlugin$1.prototype.isCtrlOrMetaPressed = function(event$1) {
		return this.isMac ? event$1.metaKey : event$1.ctrlKey;
	};
	return HyperlinkPlugin$1;
})();
function getQueryString(triggerCharacter, paragraph, previousSegment, splittedSegmentResult) {
	var result = "";
	var i$1 = paragraph.segments.indexOf(previousSegment);
	for (; i$1 >= 0; i$1--) {
		var segment = paragraph.segments[i$1];
		if (segment.segmentType != "Text") {
			result = "";
			break;
		}
		var index$1 = segment.text.lastIndexOf(triggerCharacter);
		if (index$1 >= 0) {
			result = segment.text.substring(index$1) + result;
			splittedSegmentResult === null || splittedSegmentResult === void 0 || splittedSegmentResult.unshift(index$1 > 0 ? splitTextSegment(segment, paragraph, index$1, segment.text.length) : segment);
			break;
		} else {
			result = segment.text + result;
			splittedSegmentResult === null || splittedSegmentResult === void 0 || splittedSegmentResult.unshift(segment);
		}
	}
	if (i$1 < 0) result = "";
	return result;
}
var PickerHelperImpl = function() {
	function PickerHelperImpl$1(editor, handler, triggerCharacter) {
		this.editor = editor;
		this.handler = handler;
		this.triggerCharacter = triggerCharacter;
		this.direction = null;
	}
	PickerHelperImpl$1.prototype.replaceQueryString = function(model, options, canUndoByBackspace) {
		var _this = this;
		this.editor.focus();
		formatTextSegmentBeforeSelectionMarker(this.editor, function(target, previousSegment, paragraph, _, context) {
			var potentialSegments = [];
			if (getQueryString(_this.triggerCharacter, paragraph, previousSegment, potentialSegments)) {
				potentialSegments.forEach(function(x$1) {
					return x$1.isSelected = true;
				});
				mergeModel(target, model, context);
				context.canUndoByBackspace = canUndoByBackspace;
				return true;
			} else return false;
		}, options);
	};
	PickerHelperImpl$1.prototype.closePicker = function() {
		var _a$5, _b$1;
		if (this.direction) {
			this.direction = null;
			(_b$1 = (_a$5 = this.handler).onClosePicker) === null || _b$1 === void 0 || _b$1.call(_a$5);
		}
	};
	return PickerHelperImpl$1;
}();
(function() {
	function PickerPlugin$1(triggerCharacter, handler) {
		this.triggerCharacter = triggerCharacter;
		this.handler = handler;
		this.isMac = false;
		this.lastQueryString = "";
		this.helper = null;
	}
	PickerPlugin$1.prototype.getName = function() {
		return "Picker";
	};
	PickerPlugin$1.prototype.initialize = function(editor) {
		this.isMac = !!editor.getEnvironment().isMac;
		this.helper = new PickerHelperImpl(editor, this.handler, this.triggerCharacter);
		this.handler.onInitialize(this.helper);
	};
	PickerPlugin$1.prototype.dispose = function() {
		this.handler.onDispose();
		this.helper = null;
	};
	PickerPlugin$1.prototype.willHandleEventExclusively = function(event$1) {
		var _a$5;
		return !!((_a$5 = this.helper) === null || _a$5 === void 0 ? void 0 : _a$5.direction) && event$1.eventType == "keyDown" && (isCursorMovingKey(event$1.rawEvent) || event$1.rawEvent.key == "Enter" || event$1.rawEvent.key == "Tab" || event$1.rawEvent.key == "Escape");
	};
	PickerPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (!this.helper) return;
		switch (event$1.eventType) {
			case "contentChanged":
				if (this.helper.direction) if (event$1.source == ChangeSource.SetContent) this.helper.closePicker();
				else this.onSuggestingInput(this.helper);
				break;
			case "keyDown":
				if (this.helper.direction) this.onSuggestingKeyDown(this.helper, event$1.rawEvent);
				break;
			case "input":
				if (this.helper.direction) this.onSuggestingInput(this.helper);
				else this.onInput(this.helper, event$1.rawEvent);
				break;
			case "mouseUp":
				if (this.helper.direction) this.helper.closePicker();
				break;
		}
	};
	PickerPlugin$1.prototype.onSuggestingKeyDown = function(helper, event$1) {
		var _a$5, _b$1, _c$1, _d$1, _e$1, _f$1, _g, _h, _j, _k;
		switch (event$1.key) {
			case "ArrowLeft":
			case "ArrowRight":
				if (helper.direction == "horizontal" || helper.direction == "both") {
					var isIncrement = event$1.key == "ArrowRight";
					if (helper.editor.getDOMHelper().isRightToLeft()) isIncrement = !isIncrement;
					(_b$1 = (_a$5 = this.handler).onSelectionChanged) === null || _b$1 === void 0 || _b$1.call(_a$5, isIncrement ? "next" : "previous");
				}
				event$1.preventDefault();
				break;
			case "ArrowUp":
			case "ArrowDown":
				var isIncrement = event$1.key == "ArrowDown";
				if (helper.direction != "horizontal") (_d$1 = (_c$1 = this.handler).onSelectionChanged) === null || _d$1 === void 0 || _d$1.call(_c$1, helper.direction == "both" ? isIncrement ? "nextRow" : "previousRow" : isIncrement ? "next" : "previous");
				event$1.preventDefault();
				break;
			case "PageUp":
			case "PageDown":
				(_f$1 = (_e$1 = this.handler).onSelectionChanged) === null || _f$1 === void 0 || _f$1.call(_e$1, event$1.key == "PageDown" ? "nextPage" : "previousPage");
				event$1.preventDefault();
				break;
			case "Home":
			case "End":
				var hasCtrl = this.isMac ? event$1.metaKey : event$1.ctrlKey;
				(_h = (_g = this.handler).onSelectionChanged) === null || _h === void 0 || _h.call(_g, event$1.key == "Home" ? hasCtrl ? "first" : "firstInRow" : hasCtrl ? "last" : "lastInRow");
				event$1.preventDefault();
				break;
			case "Escape":
				helper.closePicker();
				event$1.preventDefault();
				break;
			case "Enter":
			case "Tab":
				(_k = (_j = this.handler).onSelect) === null || _k === void 0 || _k.call(_j);
				event$1.preventDefault();
				break;
		}
	};
	PickerPlugin$1.prototype.onSuggestingInput = function(helper) {
		var _this = this;
		if (!formatTextSegmentBeforeSelectionMarker(helper.editor, function(_, segment, paragraph) {
			var _a$5, _b$1;
			var newQueryString = getQueryString(_this.triggerCharacter, paragraph, segment).replace(/[\u0020\u00A0]/g, " ");
			var oldQueryString = _this.lastQueryString;
			if (newQueryString && (newQueryString.length >= oldQueryString.length && newQueryString.indexOf(oldQueryString) == 0 || newQueryString.length < oldQueryString.length && oldQueryString.indexOf(newQueryString) == 0)) {
				_this.lastQueryString = newQueryString;
				(_b$1 = (_a$5 = _this.handler).onQueryStringChanged) === null || _b$1 === void 0 || _b$1.call(_a$5, newQueryString);
			} else helper.closePicker();
			return false;
		})) helper.closePicker();
	};
	PickerPlugin$1.prototype.onInput = function(helper, event$1) {
		var _this = this;
		if (event$1.inputType == "insertText" && event$1.data == this.triggerCharacter) formatTextSegmentBeforeSelectionMarker(helper.editor, function(_, segment) {
			if (segment.text.endsWith(_this.triggerCharacter)) {
				var charBeforeTrigger = segment.text[segment.text.length - 2];
				if (!charBeforeTrigger || !charBeforeTrigger.trim() || isPunctuation(charBeforeTrigger)) {
					var selection = helper.editor.getDOMSelection();
					var pos = (selection === null || selection === void 0 ? void 0 : selection.type) == "range" && selection.range.collapsed ? {
						node: selection.range.startContainer,
						offset: selection.range.startOffset
					} : null;
					if (pos) {
						_this.lastQueryString = _this.triggerCharacter;
						helper.direction = _this.handler.onTrigger(_this.lastQueryString, pos);
					}
				}
			}
			return false;
		});
	};
	return PickerPlugin$1;
})();
(function() {
	function CustomReplacePlugin$1(customReplacements) {
		this.customReplacements = customReplacements;
		this.editor = null;
		this.triggerKeys = [];
	}
	CustomReplacePlugin$1.prototype.getName = function() {
		return "CustomReplace";
	};
	CustomReplacePlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		this.triggerKeys = this.customReplacements.map(function(replacement) {
			return replacement.stringToReplace.slice(-1);
		});
	};
	CustomReplacePlugin$1.prototype.dispose = function() {
		this.editor = null;
	};
	CustomReplacePlugin$1.prototype.onPluginEvent = function(event$1) {
		if (this.editor) switch (event$1.eventType) {
			case "input":
				this.handleEditorInputEvent(this.editor, event$1);
				break;
		}
	};
	CustomReplacePlugin$1.prototype.handleEditorInputEvent = function(editor, event$1) {
		var _this = this;
		var rawEvent = event$1.rawEvent;
		var selection = editor.getDOMSelection();
		var key = rawEvent.data;
		if (this.customReplacements.length > 0 && rawEvent.inputType === "insertText" && selection && selection.type === "range" && selection.range.collapsed && key && this.triggerKeys.indexOf(key) > -1) formatTextSegmentBeforeSelectionMarker(editor, function(_model, previousSegment, paragraph, _markerFormat, context) {
			if (_this.customReplacements.some(function(_a$5) {
				var stringToReplace = _a$5.stringToReplace, replacementString = _a$5.replacementString, replacementHandler = _a$5.replacementHandler;
				return replacementHandler(previousSegment, stringToReplace, replacementString, paragraph);
			})) {
				context.canUndoByBackspace = true;
				return true;
			}
			return false;
		});
	};
	return CustomReplacePlugin$1;
})();
var RESIZE_KEYS = ["widthPx", "heightPx"];
var ROTATE_KEYS = ["angleRad"];
var CROP_KEYS = [
	"leftPercent",
	"rightPercent",
	"topPercent",
	"bottomPercent"
];
var ROTATE_CROP_KEYS = __spreadArray(__spreadArray([], __read(ROTATE_KEYS), false), __read(CROP_KEYS), false);
var ALL_KEYS = __spreadArray(__spreadArray([], __read(ROTATE_CROP_KEYS), false), __read(RESIZE_KEYS), false);
function checkEditInfoState(editInfo, compareTo) {
	if (!editInfo || !editInfo.src || ALL_KEYS.some(function(key) {
		return !isNumber(editInfo[key]);
	})) return "Invalid";
	else if (ROTATE_CROP_KEYS.every(function(key) {
		return areSameNumber(editInfo[key], 0);
	}) && !editInfo.flippedHorizontal && !editInfo.flippedVertical && (!compareTo || compareTo && editInfo.angleRad === compareTo.angleRad)) return "ResizeOnly";
	else if (compareTo && ROTATE_KEYS.every(function(key) {
		return areSameNumber(editInfo[key], compareTo[key]);
	}) && CROP_KEYS.every(function(key) {
		return areSameNumber(editInfo[key], compareTo[key]);
	}) && compareTo.flippedHorizontal === editInfo.flippedHorizontal && compareTo.flippedVertical === editInfo.flippedVertical) return "SameWithLast";
	else return "FullyChanged";
}
function isNumber(o$1) {
	return typeof o$1 === "number";
}
function areSameNumber(n1, n2) {
	return n1 != void 0 && n2 != void 0 && Math.abs(n1 - n2) < .001;
}
function getGeneratedImageSize(editInfo, beforeCrop) {
	var width = editInfo.widthPx, height = editInfo.heightPx, angleRad = editInfo.angleRad, left = editInfo.leftPercent, right = editInfo.rightPercent, top = editInfo.topPercent, bottom = editInfo.bottomPercent;
	if (height == void 0 || width == void 0 || left == void 0 || right == void 0 || top == void 0 || bottom == void 0) return;
	var angle = angleRad !== null && angleRad !== void 0 ? angleRad : 0;
	var originalWidth = width / (1 - left - right);
	var originalHeight = height / (1 - top - bottom);
	var visibleWidth = beforeCrop ? originalWidth : width;
	var visibleHeight = beforeCrop ? originalHeight : height;
	return {
		targetWidth: Math.abs(visibleWidth * Math.cos(angle)) + Math.abs(visibleHeight * Math.sin(angle)),
		targetHeight: Math.abs(visibleWidth * Math.sin(angle)) + Math.abs(visibleHeight * Math.cos(angle)),
		originalWidth,
		originalHeight,
		visibleWidth,
		visibleHeight
	};
}
function generateDataURL(image, editInfo) {
	var generatedImageSize = getGeneratedImageSize(editInfo);
	if (!generatedImageSize) return "";
	var angleRad = editInfo.angleRad, widthPx = editInfo.widthPx, heightPx = editInfo.heightPx, bottomPercent = editInfo.bottomPercent, leftPercent = editInfo.leftPercent, rightPercent = editInfo.rightPercent, topPercent = editInfo.topPercent, naturalWidth = editInfo.naturalWidth, naturalHeight = editInfo.naturalHeight;
	var angle = angleRad || 0;
	var left = leftPercent || 0;
	var right = rightPercent || 0;
	var top = topPercent || 0;
	var bottom = bottomPercent || 0;
	var nHeight = naturalHeight || image.naturalHeight;
	var nWidth = naturalWidth || image.naturalHeight;
	var width = widthPx || image.clientWidth;
	var height = heightPx || image.clientHeight;
	var imageWidth = nWidth * (1 - left - right);
	var imageHeight = nHeight * (1 - top - bottom);
	var devicePixelRatio = window.devicePixelRatio || 1;
	var canvas = document.createElement("canvas");
	var targetWidth = generatedImageSize.targetWidth, targetHeight = generatedImageSize.targetHeight;
	canvas.width = targetWidth * devicePixelRatio;
	canvas.height = targetHeight * devicePixelRatio;
	var context = canvas.getContext("2d");
	if (context) {
		context.scale(devicePixelRatio, devicePixelRatio);
		context.translate(targetWidth / 2, targetHeight / 2);
		context.rotate(angle);
		context.scale(editInfo.flippedHorizontal ? -1 : 1, editInfo.flippedVertical ? -1 : 1);
		context.drawImage(image, nWidth * left, nHeight * top, imageWidth, imageHeight, -width / 2, -height / 2, width, height);
	}
	return canvas.toDataURL("image/png", 1);
}
function getSelectedImage(model) {
	var selections = getSelectedSegmentsAndParagraphs(model, false);
	if (selections.length == 1 && selections[0][0].segmentType == "Image" && selections[0][1]) return {
		image: selections[0][0],
		paragraph: selections[0][1]
	};
	else return null;
}
function updateImageEditInfo(contentModelImage, image, newImageMetadata) {
	var contentModelMetadata = updateImageMetadata(contentModelImage, newImageMetadata !== void 0 ? function(format) {
		format = newImageMetadata;
		return format;
	} : void 0);
	return __assign(__assign({}, getInitialEditInfo(image)), contentModelMetadata);
}
function getInitialEditInfo(image) {
	return {
		src: image.getAttribute("src") || "",
		widthPx: image.clientWidth,
		heightPx: image.clientHeight,
		naturalWidth: image.naturalWidth,
		naturalHeight: image.naturalHeight,
		leftPercent: 0,
		rightPercent: 0,
		topPercent: 0,
		bottomPercent: 0,
		angleRad: 0
	};
}
function getSelectedImageMetadata(editor, image) {
	var imageMetadata = getInitialEditInfo(image);
	editor.formatContentModel(function(model) {
		var selectedImage = getSelectedImage(model);
		if (selectedImage === null || selectedImage === void 0 ? void 0 : selectedImage.image) {
			mutateSegment(selectedImage.paragraph, selectedImage === null || selectedImage === void 0 ? void 0 : selectedImage.image, function(modelImage) {
				imageMetadata = updateImageEditInfo(modelImage, image);
			});
			return true;
		}
		return false;
	});
	return imageMetadata;
}
function applyChange(editor, image, contentModelImage, editInfo, previousSrc, wasResizedOrCropped, editingImage) {
	var _a$5;
	var newSrc = "";
	var imageEditing = editingImage !== null && editingImage !== void 0 ? editingImage : image;
	var state$1 = checkEditInfoState(editInfo, (_a$5 = updateImageEditInfo(contentModelImage, imageEditing)) !== null && _a$5 !== void 0 ? _a$5 : void 0);
	switch (state$1) {
		case "ResizeOnly":
			newSrc = editInfo.src || "";
			break;
		case "SameWithLast":
			newSrc = previousSrc;
			break;
		case "FullyChanged":
			newSrc = generateDataURL(editingImage !== null && editingImage !== void 0 ? editingImage : image, editInfo);
			break;
	}
	if (newSrc != previousSrc) newSrc = editor.triggerEvent("editImage", {
		image,
		originalSrc: editInfo.src || image.src,
		previousSrc,
		newSrc
	}).newSrc;
	if (newSrc == editInfo.src) updateImageEditInfo(contentModelImage, imageEditing, null);
	else updateImageEditInfo(contentModelImage, imageEditing, editInfo);
	var generatedImageSize = getGeneratedImageSize(editInfo);
	if (generatedImageSize) {
		contentModelImage.src = newSrc;
		if (wasResizedOrCropped || state$1 == "FullyChanged") {
			contentModelImage.format.width = generatedImageSize.targetWidth + "px";
			contentModelImage.format.height = generatedImageSize.targetHeight + "px";
		}
	}
	return state$1;
}
function canRegenerateImage(img) {
	if (!img) return false;
	try {
		var canvas = img.ownerDocument.createElement("canvas");
		canvas.width = 10;
		canvas.height = 10;
		var context = canvas.getContext("2d");
		if (context) {
			context.drawImage(img, 0, 0);
			context.getImageData(0, 0, 1, 1);
			return true;
		}
		return false;
	} catch (_a$5) {
		return false;
	}
}
var DEG_PER_RAD = 180 / Math.PI;
var ROTATION = {
	sw: 0,
	nw: 90,
	ne: 180,
	se: 270
};
var Xs = [
	"w",
	"",
	"e"
];
var Ys = [
	"s",
	"",
	"n"
];
var ROTATE_HANDLE_TOP = 21;
var XS_CROP = ["w", "e"];
var YS_CROP = ["s", "n"];
function getPx(value) {
	return value + "px";
}
function isASmallImage(widthPx, heightPx) {
	return widthPx && heightPx && (widthPx < 42 || heightPx < 42) ? true : false;
}
function rotateCoordinate(x$1, y$1, angle) {
	if (x$1 == 0 && y$1 == 0) return [0, 0];
	var hypotenuse = Math.sqrt(x$1 * x$1 + y$1 * y$1);
	angle = Math.atan2(y$1, x$1) - angle;
	return [hypotenuse * Math.cos(angle), hypotenuse * Math.sin(angle)];
}
function setFlipped(element, flippedHorizontally, flippedVertically) {
	if (element) element.style.transform = "scale(" + (flippedHorizontally ? -1 : 1) + ", " + (flippedVertically ? -1 : 1) + ")";
}
function setWrapperSizeDimensions(wrapper, image, width, height, isRotating) {
	if (image.style.borderStyle) {
		var borderWidth = image.style.borderWidth ? 2 * parseInt(image.style.borderWidth) : 2;
		if (isRotating) {
			wrapper.style.width = getPx(parseInt(image.style.width) + borderWidth);
			wrapper.style.height = getPx(parseInt(image.style.height) + borderWidth);
			return;
		}
		wrapper.style.width = getPx(width + borderWidth);
		wrapper.style.height = getPx(height + borderWidth);
		return;
	}
	wrapper.style.width = getPx(width);
	wrapper.style.height = getPx(height);
}
function setSize(element, left, top, right, bottom, width, height) {
	element.style.left = left !== void 0 ? getPx(left) : element.style.left;
	element.style.top = top !== void 0 ? getPx(top) : element.style.top;
	element.style.right = right !== void 0 ? getPx(right) : element.style.right;
	element.style.bottom = bottom !== void 0 ? getPx(bottom) : element.style.bottom;
	element.style.width = width !== void 0 ? getPx(width) : element.style.width;
	element.style.height = height !== void 0 ? getPx(height) : element.style.height;
}
function checkIfImageWasResized(image) {
	var style = image.style;
	if ((style.maxWidth === "" || style.maxWidth === "initial" || style.maxWidth === "auto") && (isFixedNumberValue(style.height) || isFixedNumberValue(style.width))) return true;
	else return false;
}
function isFixedNumberValue(value) {
	var numberValue = typeof value === "string" ? parseInt(value) : value;
	return !isNaN(numberValue);
}
function getActualWrapperDimensions(image, wrapperWidth, wrapperHeight) {
	var hasBorder = image.style.borderStyle;
	var borderWidth = hasBorder && image.style.borderWidth ? 2 * parseInt(image.style.borderWidth) : hasBorder ? 2 : 0;
	return {
		width: wrapperWidth - borderWidth,
		height: wrapperHeight - borderWidth
	};
}
var ImageEditElementClass;
(function(ImageEditElementClass$1) {
	ImageEditElementClass$1["ResizeHandle"] = "r_resizeH";
	ImageEditElementClass$1["RotateHandle"] = "r_rotateH";
	ImageEditElementClass$1["RotateCenter"] = "r_rotateC";
	ImageEditElementClass$1["CropOverlay"] = "r_cropO";
	ImageEditElementClass$1["CropContainer"] = "r_cropC";
	ImageEditElementClass$1["CropHandle"] = "r_cropH";
})(ImageEditElementClass || (ImageEditElementClass = {}));
function createImageCropper(doc) {
	return getCropHTML().map(function(data) {
		var cropper = createElement(data, doc);
		if (cropper && isNodeOfType(cropper, "ELEMENT_NODE") && isElementOfType(cropper, "div")) return cropper;
	}).filter(function(cropper) {
		return !!cropper;
	});
}
function getCropHTML() {
	var overlayHTML = {
		tag: "div",
		style: "position:absolute;background-color:rgb(0,0,0,0.5);pointer-events:none",
		className: ImageEditElementClass.CropOverlay
	};
	var containerHTML = {
		tag: "div",
		style: "position:absolute;overflow:hidden;inset:0px;",
		className: ImageEditElementClass.CropContainer,
		children: []
	};
	if (containerHTML) XS_CROP.forEach(function(x$1) {
		return YS_CROP.forEach(function(y$1) {
			var _a$5;
			return (_a$5 = containerHTML.children) === null || _a$5 === void 0 ? void 0 : _a$5.push(getCropHTMLInternal(x$1, y$1));
		});
	});
	return [
		containerHTML,
		overlayHTML,
		overlayHTML,
		overlayHTML,
		overlayHTML
	];
}
function getCropHTMLInternal(x$1, y$1) {
	var leftOrRight = x$1 == "w" ? "left" : "right";
	var topOrBottom = y$1 == "n" ? "top" : "bottom";
	var rotation = ROTATION[y$1 + x$1];
	return {
		tag: "div",
		className: ImageEditElementClass.CropHandle,
		style: "position:absolute;pointer-events:auto;cursor:" + y$1 + x$1 + "-resize;" + leftOrRight + ":0;" + topOrBottom + ":0;width:22px;height:22px;transform:rotate(" + rotation + "deg)",
		dataset: {
			x: x$1,
			y: y$1
		},
		children: getCropHandleHTML()
	};
}
function getCropHandleHTML() {
	var result = [];
	[0, 1].forEach(function(layer) {
		return [0, 1].forEach(function(dir) {
			result.push(getCropHandleHTMLInternal(layer, dir));
		});
	});
	return result;
}
function getCropHandleHTMLInternal(layer, dir) {
	var position = dir == 0 ? "right:" + layer + "px;height:" + (7 - layer * 2) + "px;" : "top:" + layer + "px;width:" + (7 - layer * 2) + "px;";
	var bgColor = layer == 0 ? "white" : "black";
	return {
		tag: "div",
		style: "position:absolute;left:" + layer + "px;bottom:" + layer + "px;" + position + ";background-color:" + bgColor
	};
}
var RESIZE_HANDLE_MARGIN = 6;
var RESIZE_HANDLE_SIZE = 10;
function createImageResizer(doc, disableSideResize, onShowResizeHandle) {
	var cornerElements = getCornerResizeHTML(onShowResizeHandle);
	var sideElements = getSideResizeHTML(onShowResizeHandle);
	return (disableSideResize ? cornerElements : cornerElements.concat(sideElements)).map(function(element) {
		var handle = createElement(element, doc);
		if (isNodeOfType(handle, "ELEMENT_NODE") && isElementOfType(handle, "div")) return handle;
	}).filter(function(element) {
		return !!element;
	});
}
function getCornerResizeHTML(onShowResizeHandle) {
	var result = [];
	Xs.forEach(function(x$1) {
		return Ys.forEach(function(y$1) {
			var elementData = x$1 == "" == (y$1 == "") ? getResizeHandleHTML(x$1, y$1) : null;
			if (onShowResizeHandle && elementData) onShowResizeHandle(elementData, x$1, y$1);
			if (elementData) result.push(elementData);
		});
	});
	return result;
}
function getSideResizeHTML(onShowResizeHandle) {
	var result = [];
	Xs.forEach(function(x$1) {
		return Ys.forEach(function(y$1) {
			var elementData = x$1 == "" != (y$1 == "") ? getResizeHandleHTML(x$1, y$1) : null;
			if (onShowResizeHandle && elementData) onShowResizeHandle(elementData, x$1, y$1);
			if (elementData) result.push(elementData);
		});
	});
	return result;
}
var createHandleStyle = function(direction, topOrBottom, leftOrRight) {
	return "position:relative;width:" + RESIZE_HANDLE_SIZE + "px;height:" + RESIZE_HANDLE_SIZE + "px;background-color: #FFFFFF;cursor:" + direction + "-resize;" + topOrBottom + ":-" + RESIZE_HANDLE_MARGIN + "px;" + leftOrRight + ":-" + RESIZE_HANDLE_MARGIN + "px;border-radius:100%;border: 2px solid #bfbfbf;box-shadow: 0px 0.36316px 1.36185px rgba(100, 100, 100, 0.25);";
};
function getResizeHandleHTML(x$1, y$1) {
	var leftOrRight = x$1 == "w" ? "left" : "right";
	var topOrBottom = y$1 == "n" ? "top" : "bottom";
	var leftOrRightValue = x$1 == "" ? "50%" : "0px";
	var topOrBottomValue = y$1 == "" ? "50%" : "0px";
	var direction = y$1 + x$1;
	return x$1 == "" && y$1 == "" ? null : {
		tag: "div",
		style: "position:absolute;" + leftOrRight + ":" + leftOrRightValue + ";" + topOrBottom + ":" + topOrBottomValue,
		children: [{
			tag: "div",
			style: createHandleStyle(direction, topOrBottom, leftOrRight),
			className: ImageEditElementClass.ResizeHandle,
			dataset: {
				x: x$1,
				y: y$1
			}
		}]
	};
}
function createImageRotator(doc, htmlOptions) {
	return getRotateHTML(htmlOptions).map(function(element) {
		var rotator = createElement(element, doc);
		if (isNodeOfType(rotator, "ELEMENT_NODE") && isElementOfType(rotator, "div")) return rotator;
	}).filter(function(rotator) {
		return !!rotator;
	});
}
function getRotateHTML(_a$5) {
	var borderColor = _a$5.borderColor, rotateHandleBackColor = _a$5.rotateHandleBackColor, disableSideResize = _a$5.disableSideResize;
	return [{
		tag: "div",
		className: ImageEditElementClass.RotateCenter,
		style: "position:absolute;left:50%;width:1px;background-color:" + borderColor + ";top:" + (disableSideResize ? -15 : -ROTATE_HANDLE_TOP) + "px;height:15px;margin-left:-1px;",
		children: [{
			tag: "div",
			className: ImageEditElementClass.RotateHandle,
			style: "position:absolute;background-color:" + rotateHandleBackColor + ";border:solid 1px " + borderColor + ";border-radius:50%;width:32px;height:32px;left:-17px;cursor:move;top:-32px;line-height: 0px;",
			children: [getRotateIconHTML(borderColor)]
		}]
	}];
}
function getRotateIconHTML(borderColor) {
	var _a$5;
	return {
		tag: "svg",
		namespace: "http://www.w3.org/2000/svg",
		style: "width:16px;height:16px;margin: 8px 8px",
		children: [{
			tag: "path",
			namespace: "http://www.w3.org/2000/svg",
			attributes: (_a$5 = {
				d: "M 10.5,10.0 A 3.8,3.8 0 1 1 6.7,6.3",
				transform: "matrix(1.1 1.1 -1.1 1.1 11.6 -10.8)"
			}, _a$5["fill-opacity"] = "0", _a$5.stroke = borderColor, _a$5)
		}, {
			tag: "path",
			namespace: "http://www.w3.org/2000/svg",
			attributes: {
				d: "M12.0 3.648l.884-.884.53 2.298-2.298-.53z",
				stroke: borderColor
			}
		}]
	};
}
var IMAGE_EDIT_SHADOW_ROOT = "ImageEditShadowRoot";
function createImageWrapper(editor, image, options, editInfo, htmlOptions, operation) {
	var imageClone = cloneImage(image, editInfo);
	var doc = editor.getDocument();
	var rotators = [];
	if (!options.disableRotate && operation.indexOf("rotate") > -1) rotators = createImageRotator(doc, htmlOptions);
	var resizers = [];
	if (operation.indexOf("resize") > -1) resizers = createImageResizer(doc, !!options.disableSideResize);
	var croppers = [];
	if (operation.indexOf("crop") > -1) croppers = createImageCropper(doc);
	var wrapper = createWrapper(editor, imageClone, options, editInfo, resizers, rotators, croppers);
	return {
		wrapper,
		shadowSpan: createShadowSpan(wrapper, wrap(doc, image, "span")),
		imageClone,
		resizers,
		rotators,
		croppers
	};
}
var createShadowSpan = function(wrapper, imageSpan) {
	var shadowRoot = imageSpan.attachShadow({ mode: "open" });
	imageSpan.id = IMAGE_EDIT_SHADOW_ROOT;
	shadowRoot.appendChild(wrapper);
	return imageSpan;
};
var createWrapper = function(editor, image, options, editInfo, resizers, rotators, cropper) {
	var _a$5;
	var doc = editor.getDocument();
	var wrapper = doc.createElement("span");
	var imageBox = doc.createElement("div");
	imageBox.setAttribute("style", "position:relative;width:100%;height:100%;overflow:hidden;transform:scale(1);");
	imageBox.appendChild(image);
	wrapper.setAttribute("style", "font-size: 24px; margin: 0px; transform: rotate(" + ((_a$5 = editInfo.angleRad) !== null && _a$5 !== void 0 ? _a$5 : 0) + "rad);");
	wrapper.style.display = editor.getEnvironment().isSafari ? "-webkit-inline-flex" : "inline-flex";
	var border = createBorder(editor, options.borderColor);
	wrapper.appendChild(imageBox);
	wrapper.appendChild(border);
	wrapper.style.userSelect = "none";
	if (resizers && (resizers === null || resizers === void 0 ? void 0 : resizers.length) > 0) resizers.forEach(function(resizer) {
		wrapper.appendChild(resizer);
	});
	if (rotators && rotators.length > 0) rotators.forEach(function(r$1) {
		wrapper.appendChild(r$1);
	});
	if (cropper && cropper.length > 0) cropper.forEach(function(c$1) {
		wrapper.appendChild(c$1);
	});
	return wrapper;
};
var createBorder = function(editor, borderColor) {
	var resizeBorder = editor.getDocument().createElement("div");
	resizeBorder.setAttribute("style", "position:absolute;left:0;right:0;top:0;bottom:0;border:solid 2px " + borderColor + ";pointer-events:none;");
	return resizeBorder;
};
var cloneImage = function(image, editInfo) {
	var imageClone = image.cloneNode(true);
	imageClone.style.removeProperty("transform");
	if (editInfo.src) {
		imageClone.src = editInfo.src;
		imageClone.removeAttribute("id");
		imageClone.style.removeProperty("max-width");
		imageClone.style.removeProperty("max-height");
		imageClone.style.width = editInfo.widthPx + "px";
		imageClone.style.height = editInfo.heightPx + "px";
	}
	return imageClone;
};
var Cropper = {
	onDragStart: function(_a$5) {
		var editInfo = _a$5.editInfo;
		return __assign({}, editInfo);
	},
	onDragging: function(_a$5, e$1, base, dx, dy) {
		var _b$1;
		var _c$1;
		var editInfo = _a$5.editInfo, x$1 = _a$5.x, y$1 = _a$5.y, options = _a$5.options;
		_b$1 = __read(rotateCoordinate(dx, dy, (_c$1 = editInfo.angleRad) !== null && _c$1 !== void 0 ? _c$1 : 0), 2), dx = _b$1[0], dy = _b$1[1];
		var widthPx = editInfo.widthPx, heightPx = editInfo.heightPx, leftPercent = editInfo.leftPercent, rightPercent = editInfo.rightPercent, topPercent = editInfo.topPercent, bottomPercent = editInfo.bottomPercent;
		if (leftPercent === void 0 || rightPercent === void 0 || topPercent === void 0 || bottomPercent === void 0 || base.leftPercent === void 0 || base.rightPercent === void 0 || base.topPercent === void 0 || base.bottomPercent === void 0 || widthPx === void 0 || heightPx === void 0) return false;
		var minWidth = options.minWidth, minHeight = options.minHeight;
		var widthPercent = 1 - leftPercent - rightPercent;
		var heightPercent = 1 - topPercent - bottomPercent;
		if (widthPercent > 0 && heightPercent > 0 && minWidth !== void 0 && minHeight !== void 0) {
			var fullWidth = widthPx / widthPercent;
			var fullHeight = heightPx / heightPercent;
			var newLeft = x$1 != "e" ? crop(base.leftPercent, dx, fullWidth, rightPercent, minWidth) : leftPercent;
			var newRight = x$1 != "w" ? crop(base.rightPercent, -dx, fullWidth, leftPercent, minWidth) : rightPercent;
			var newTop = y$1 != "s" ? crop(base.topPercent, dy, fullHeight, bottomPercent, minHeight) : topPercent;
			var newBottom = y$1 != "n" ? crop(base.bottomPercent, -dy, fullHeight, topPercent, minHeight) : bottomPercent;
			editInfo.leftPercent = newLeft;
			editInfo.rightPercent = newRight;
			editInfo.topPercent = newTop;
			editInfo.bottomPercent = newBottom;
			editInfo.widthPx = fullWidth * (1 - newLeft - newRight);
			editInfo.heightPx = fullHeight * (1 - newTop - newBottom);
			return true;
		} else return false;
	}
};
function crop(basePercentage, deltaValue, fullValue, currentPercentage, minValue) {
	var maxValue = fullValue * (1 - currentPercentage) - minValue;
	var newValue = fullValue * basePercentage + deltaValue;
	return Math.max(Math.min(newValue, maxValue), 0) / fullValue;
}
function findEditingImage(group, imageId) {
	var imageAndParagraph = null;
	queryContentModelBlocks(group, "Paragraph", function(paragraph) {
		var e_1, _a$5;
		try {
			for (var _b$1 = __values(paragraph.segments), _c$1 = _b$1.next(); !_c$1.done; _c$1 = _b$1.next()) {
				var segment = _c$1.value;
				if (segment.segmentType == "Image" && (imageId && segment.format.id == imageId || segment.format.imageState == "isEditing")) {
					imageAndParagraph = {
						image: segment,
						paragraph
					};
					return true;
				}
			}
		} catch (e_1_1) {
			e_1 = { error: e_1_1 };
		} finally {
			try {
				if (_c$1 && !_c$1.done && (_a$5 = _b$1.return)) _a$5.call(_b$1);
			} finally {
				if (e_1) throw e_1.error;
			}
		}
		return false;
	}, true);
	return imageAndParagraph;
}
function filterInnerResizerHandles(resizeHandles) {
	return resizeHandles.map(function(resizer) {
		var resizeHandle = resizer.firstElementChild;
		if (isNodeOfType(resizeHandle, "ELEMENT_NODE") && isElementOfType(resizeHandle, "div")) return resizeHandle;
	}).filter(function(handle) {
		return !!handle;
	});
}
function getDropAndDragHelpers(wrapper, editInfo, options, elementClass, helper, updateWrapper$1, zoomScale, useTouch) {
	return getEditElements(wrapper, elementClass).map(function(element) {
		return new DragAndDropHelper(element, {
			editInfo,
			options,
			elementClass,
			x: element.dataset.x,
			y: element.dataset.y
		}, updateWrapper$1, helper, zoomScale, useTouch);
	});
}
function getEditElements(wrapper, elementClass) {
	return toArray(wrapper.querySelectorAll("." + elementClass));
}
var LIGHT_MODE_BGCOLOR = "white";
var DARK_MODE_BGCOLOR = "#333";
var getHTMLImageOptions = function(editor, options, editInfo) {
	var _a$5, _b$1;
	return {
		borderColor: options.borderColor || (editor.isDarkMode() ? DARK_MODE_BGCOLOR : LIGHT_MODE_BGCOLOR),
		rotateHandleBackColor: editor.isDarkMode() ? DARK_MODE_BGCOLOR : LIGHT_MODE_BGCOLOR,
		isSmallImage: isASmallImage((_a$5 = editInfo.widthPx) !== null && _a$5 !== void 0 ? _a$5 : 0, (_b$1 = editInfo.heightPx) !== null && _b$1 !== void 0 ? _b$1 : 0),
		disableSideResize: !!options.disableSideResize
	};
};
function normalizeImageSelection(imageAndParagraph) {
	var paragraph = imageAndParagraph.paragraph;
	var index$1 = paragraph.segments.indexOf(imageAndParagraph.image);
	if (index$1 > 0) {
		var markerBefore = paragraph.segments[index$1 - 1];
		var markerAfter = paragraph.segments[index$1 + 1];
		if (markerBefore && markerAfter && markerAfter.segmentType == "SelectionMarker" && markerBefore.segmentType == "SelectionMarker" && markerAfter.isSelected && markerBefore.isSelected) {
			var mutatedParagraph = mutateBlock(paragraph);
			mutatedParagraph.segments.splice(index$1 - 1, 1);
			mutatedParagraph.segments.splice(index$1, 1);
		}
		return imageAndParagraph;
	}
}
var Resizer = {
	onDragStart: function(_a$5) {
		var editInfo = _a$5.editInfo;
		return __assign({}, editInfo);
	},
	onDragging: function(_a$5, e$1, base, deltaX, deltaY) {
		var _b$1;
		var _c$1;
		var x$1 = _a$5.x, y$1 = _a$5.y, editInfo = _a$5.editInfo, options = _a$5.options;
		if (base.heightPx && base.widthPx && options.minWidth !== void 0 && options.minHeight !== void 0) {
			var ratio = base.widthPx > 0 && base.heightPx > 0 ? base.widthPx * 1 / base.heightPx : 0;
			_b$1 = __read(rotateCoordinate(deltaX, deltaY, (_c$1 = editInfo.angleRad) !== null && _c$1 !== void 0 ? _c$1 : 0), 2), deltaX = _b$1[0], deltaY = _b$1[1];
			var horizontalOnly = x$1 == "";
			var verticalOnly = y$1 == "";
			var shouldPreserveRatio = !(horizontalOnly || verticalOnly) && (options.preserveRatio || e$1.shiftKey);
			var newWidth = horizontalOnly ? base.widthPx : Math.max(base.widthPx + deltaX * (x$1 == "w" ? -1 : 1), options.minWidth);
			var newHeight = verticalOnly ? base.heightPx : Math.max(base.heightPx + deltaY * (y$1 == "n" ? -1 : 1), options.minHeight);
			if (shouldPreserveRatio && ratio > 0) if (ratio > 1) {
				newWidth = newHeight * ratio;
				if (newWidth < options.minWidth) {
					newWidth = options.minWidth;
					newHeight = newWidth / ratio;
				}
			} else {
				newHeight = newWidth / ratio;
				if (newHeight < options.minHeight) {
					newHeight = options.minHeight;
					newWidth = newHeight * ratio;
				}
			}
			editInfo.widthPx = newWidth;
			editInfo.heightPx = newHeight;
			return true;
		} else return false;
	}
};
var Rotator = {
	onDragStart: function(_a$5) {
		var editInfo = _a$5.editInfo;
		return __assign({}, editInfo);
	},
	onDragging: function(_a$5, e$1, base, deltaX, deltaY) {
		var _b$1, _c$1;
		var editInfo = _a$5.editInfo, options = _a$5.options;
		if (editInfo.heightPx) {
			var distance = editInfo.heightPx / 2 + 31;
			var newX = distance * Math.sin((_b$1 = base.angleRad) !== null && _b$1 !== void 0 ? _b$1 : 0) + deltaX;
			var newY = distance * Math.cos((_c$1 = base.angleRad) !== null && _c$1 !== void 0 ? _c$1 : 0) - deltaY;
			var angleInRad = Math.atan2(newX, newY);
			if (!e$1.altKey && options && options.minRotateDeg !== void 0) {
				var angleInDeg = angleInRad * DEG_PER_RAD;
				angleInRad = Math.round(angleInDeg / options.minRotateDeg) * options.minRotateDeg / DEG_PER_RAD;
			}
			if (editInfo.angleRad != angleInRad) {
				editInfo.angleRad = angleInRad;
				return true;
			}
		}
		return false;
	}
};
var PI = Math.PI;
var DIRECTIONS = 8;
var DirectionRad = PI * 2 / DIRECTIONS;
var DirectionOrder = [
	"nw",
	"n",
	"ne",
	"e",
	"se",
	"s",
	"sw",
	"w"
];
function handleRadIndexCalculator(angleRad) {
	var idx = Math.round(angleRad / DirectionRad) % DIRECTIONS;
	return idx < 0 ? idx + DIRECTIONS : idx;
}
function rotateHandles(angleRad, y$1, x$1) {
	if (y$1 === void 0) y$1 = "";
	if (x$1 === void 0) x$1 = "";
	var radIndex = handleRadIndexCalculator(angleRad);
	var originalDirection = y$1 + x$1;
	var originalIndex = DirectionOrder.indexOf(originalDirection);
	var rotatedIndex = originalIndex >= 0 && originalIndex + radIndex;
	return rotatedIndex ? DirectionOrder[rotatedIndex % DIRECTIONS] : "";
}
function updateHandleCursor(handles, angleRad) {
	handles.forEach(function(handle) {
		var _a$5 = handle.dataset, y$1 = _a$5.y, x$1 = _a$5.x;
		handle.style.cursor = rotateHandles(angleRad, y$1, x$1) + "-resize";
	});
}
function updateRotateHandle(editorRect, angleRad, wrapper, rotateCenter, rotateHandle, isSmallImage, disableSideResize) {
	if (isSmallImage) {
		rotateCenter.style.display = "none";
		rotateHandle.style.display = "none";
		return;
	} else {
		rotateCenter.style.display = "";
		rotateHandle.style.display = "";
		var rotateCenterRect = rotateCenter.getBoundingClientRect();
		var wrapperRect = wrapper.getBoundingClientRect();
		var ROTATOR_HEIGHT = 53;
		if (rotateCenterRect && wrapperRect) {
			var adjustedDistance = Number.MAX_SAFE_INTEGER;
			var angle = angleRad * DEG_PER_RAD;
			if (angle < 45 && angle > -45 && wrapperRect.top - editorRect.top < ROTATOR_HEIGHT) adjustedDistance = rotateCenterRect.top - editorRect.top;
			else if (angle <= -80 && angle >= -100 && wrapperRect.left - editorRect.left < ROTATOR_HEIGHT) adjustedDistance = rotateCenterRect.left - editorRect.left;
			else if (angle >= 80 && angle <= 100 && editorRect.right - wrapperRect.right < ROTATOR_HEIGHT) {
				var right = rotateCenterRect.right - editorRect.right;
				adjustedDistance = Math.min(editorRect.right - wrapperRect.right, right);
			} else if ((angle <= -160 || angle >= 160) && editorRect.bottom - wrapperRect.bottom < ROTATOR_HEIGHT) {
				var bottom = rotateCenterRect.bottom - editorRect.bottom;
				adjustedDistance = Math.min(editorRect.bottom - wrapperRect.bottom, bottom);
			}
			var rotateGap = Math.max(Math.min(15, adjustedDistance), 0);
			var rotateTop = Math.max(Math.min(32, adjustedDistance - rotateGap), 0);
			rotateCenter.style.top = -rotateGap - (disableSideResize ? 0 : 6) + "px";
			rotateCenter.style.height = rotateGap + "px";
			rotateHandle.style.top = -rotateTop + "px";
		}
	}
}
function doubleCheckResize(editInfo, preserveRatio, actualWidth, actualHeight) {
	var widthPx = editInfo.widthPx, heightPx = editInfo.heightPx;
	if (widthPx == void 0 || heightPx == void 0) return;
	var ratio = heightPx > 0 ? widthPx / heightPx : 0;
	actualWidth = Math.floor(actualWidth);
	actualHeight = Math.floor(actualHeight);
	widthPx = Math.floor(widthPx);
	heightPx = Math.floor(heightPx);
	editInfo.widthPx = actualWidth;
	editInfo.heightPx = actualHeight;
	if (preserveRatio && ratio > 0 && (widthPx !== actualWidth || heightPx !== actualHeight)) if (actualWidth < widthPx) editInfo.heightPx = actualWidth / ratio;
	else editInfo.widthPx = actualHeight * ratio;
}
function updateSideHandlesVisibility(handles, isSmall) {
	handles.forEach(function(handle) {
		var _a$5 = handle.dataset, y$1 = _a$5.y, x$1 = _a$5.x;
		var coordinate = (y$1 !== null && y$1 !== void 0 ? y$1 : "") + (x$1 !== null && x$1 !== void 0 ? x$1 : "");
		var isSideHandle = [
			"n",
			"s",
			"e",
			"w"
		].indexOf(coordinate) > -1;
		handle.style.display = isSideHandle && isSmall ? "none" : "";
	});
}
function updateWrapper(editInfo, options, image, clonedImage, wrapper, resizers, croppers, isRTL, isRotating) {
	var angleRad = editInfo.angleRad, bottomPercent = editInfo.bottomPercent, leftPercent = editInfo.leftPercent, rightPercent = editInfo.rightPercent, topPercent = editInfo.topPercent, flippedHorizontal = editInfo.flippedHorizontal, flippedVertical = editInfo.flippedVertical;
	var generateImageSize = getGeneratedImageSize(editInfo, croppers && (croppers === null || croppers === void 0 ? void 0 : croppers.length) > 0);
	if (!generateImageSize) return;
	var targetWidth = generateImageSize.targetWidth, targetHeight = generateImageSize.targetHeight, originalWidth = generateImageSize.originalWidth, originalHeight = generateImageSize.originalHeight, visibleWidth = generateImageSize.visibleWidth, visibleHeight = generateImageSize.visibleHeight;
	var marginHorizontal = (targetWidth - visibleWidth) / 2;
	var marginVertical = (targetHeight - visibleHeight) / 2;
	var cropLeftPx = originalWidth * (leftPercent || 0);
	var cropRightPx = originalWidth * (rightPercent || 0);
	var cropTopPx = originalHeight * (topPercent || 0);
	var cropBottomPx = originalHeight * (bottomPercent || 0);
	wrapper.style.marginTop = marginVertical + "px";
	wrapper.style.marginBottom = marginVertical + 5 + "px ";
	wrapper.style.marginLeft = marginHorizontal + "px";
	wrapper.style.marginRight = marginHorizontal + "px";
	wrapper.style.transform = "rotate(" + angleRad + "rad)";
	setWrapperSizeDimensions(wrapper, image, visibleWidth, visibleHeight, !!isRotating);
	wrapper.style.verticalAlign = "text-bottom";
	if (isRTL) {
		wrapper.style.textAlign = "right";
		if (!croppers) {
			clonedImage.style.left = getPx(cropLeftPx);
			clonedImage.style.right = getPx(-cropRightPx);
		}
	} else wrapper.style.textAlign = "left";
	if (!isRotating) {
		clonedImage.style.width = getPx(originalWidth);
		clonedImage.style.height = getPx(originalHeight);
	}
	clonedImage.style.position = "absolute";
	setFlipped(clonedImage.parentElement, flippedHorizontal, flippedVertical);
	var smallImage = isASmallImage(visibleWidth, visibleWidth);
	if (!croppers) clonedImage.style.margin = -cropTopPx + "px 0 0 " + -cropLeftPx + "px";
	if (croppers && croppers.length > 0) {
		var cropContainer = croppers[0];
		var cropOverlays = croppers.filter(function(cropper) {
			return cropper.className === ImageEditElementClass.CropOverlay;
		});
		var cropHandles = toArray(cropContainer.querySelectorAll("." + ImageEditElementClass.CropHandle));
		setSize(cropContainer, cropLeftPx, cropTopPx, cropRightPx, cropBottomPx, void 0, void 0);
		setSize(cropOverlays[0], 0, 0, cropRightPx, void 0, void 0, cropTopPx);
		setSize(cropOverlays[1], void 0, 0, 0, cropBottomPx, cropRightPx, void 0);
		setSize(cropOverlays[2], cropLeftPx, void 0, 0, 0, void 0, cropBottomPx);
		setSize(cropOverlays[3], 0, cropTopPx, void 0, 0, cropLeftPx, void 0);
		if (angleRad !== void 0) updateHandleCursor(cropHandles, angleRad);
	}
	if (resizers && !isRotating) {
		var clientWidth = wrapper.clientWidth;
		var clientHeight = wrapper.clientHeight;
		var actualDimensions = getActualWrapperDimensions(image, clientWidth, clientHeight);
		doubleCheckResize(editInfo, options.preserveRatio || false, actualDimensions.width, actualDimensions.height);
		var resizeHandles = filterInnerResizerHandles(resizers);
		if (angleRad !== void 0) updateHandleCursor(resizeHandles, angleRad);
		updateSideHandlesVisibility(resizeHandles, smallImage);
	}
}
var DefaultOptions = {
	borderColor: "#DB626C",
	minWidth: 10,
	minHeight: 10,
	preserveRatio: true,
	disableRotate: false,
	disableSideResize: false,
	onSelectState: ["resize", "rotate"]
};
var MouseRightButton = 2;
var DRAG_ID = "_dragging";
var IMAGE_EDIT_CLASS = "imageEdit";
var IMAGE_EDIT_CLASS_CARET = "imageEditCaretColor";
var IMAGE_EDIT_FORMAT_EVENT = "ImageEditEvent";
(function() {
	function ImageEditPlugin$1(options) {
		this.editor = null;
		this.shadowSpan = null;
		this.selectedImage = null;
		this.wrapper = null;
		this.imageEditInfo = null;
		this.imageHTMLOptions = null;
		this.dndHelpers = [];
		this.clonedImage = null;
		this.lastSrc = null;
		this.wasImageResized = false;
		this.isCropMode = false;
		this.resizers = [];
		this.rotators = [];
		this.croppers = [];
		this.zoomScale = 1;
		this.disposer = null;
		this.isEditing = false;
		this.options = __assign(__assign({}, DefaultOptions), options);
	}
	ImageEditPlugin$1.prototype.getName = function() {
		return "ImageEdit";
	};
	ImageEditPlugin$1.prototype.initialize = function(editor) {
		var _this = this;
		this.editor = editor;
		this.disposer = editor.attachDomEvent({
			blur: { beforeDispatch: function() {
				if (_this.isEditing && _this.editor && !_this.editor.isDisposed()) _this.applyFormatWithContentModel(_this.editor, _this.isCropMode, true);
			} },
			dragstart: { beforeDispatch: function(ev) {
				if (_this.editor) {
					var target = ev.target;
					if (_this.isImageSelection(target)) target.id = target.id + DRAG_ID;
				}
			} },
			dragend: { beforeDispatch: function(ev) {
				if (_this.editor) {
					var target = ev.target;
					if (_this.isImageSelection(target) && target.id.includes(DRAG_ID)) target.id = target.id.replace(DRAG_ID, "").trim();
				}
			} }
		});
	};
	ImageEditPlugin$1.prototype.dispose = function() {
		if (this.disposer) {
			this.disposer();
			this.disposer = null;
		}
		this.isEditing = false;
		this.cleanInfo();
		this.editor = null;
	};
	ImageEditPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (!this.editor) return;
		switch (event$1.eventType) {
			case "mouseDown":
				this.mouseDownHandler(this.editor, event$1);
				break;
			case "mouseUp":
				this.mouseUpHandler(this.editor, event$1);
				break;
			case "keyDown":
				this.keyDownHandler(this.editor, event$1);
				break;
			case "contentChanged":
				this.contentChangedHandler(this.editor, event$1);
				break;
			case "extractContentWithDom":
				this.removeImageEditing(event$1.clonedRoot);
				break;
			case "beforeLogicalRootChange":
				this.handleBeforeLogicalRootChange();
				break;
		}
	};
	ImageEditPlugin$1.prototype.handleBeforeLogicalRootChange = function() {
		if (this.isEditing && this.editor && !this.editor.isDisposed()) {
			this.applyFormatWithContentModel(this.editor, this.isCropMode, false);
			this.removeImageWrapper();
			this.cleanInfo();
		}
	};
	ImageEditPlugin$1.prototype.removeImageEditing = function(clonedRoot) {
		clonedRoot.querySelectorAll("img").forEach(function(image) {
			if (image.dataset.editingInfo) delete image.dataset.editingInfo;
		});
	};
	ImageEditPlugin$1.prototype.isImageSelection = function(target) {
		return isNodeOfType(target, "ELEMENT_NODE") && (isElementOfType(target, "img") || !!(isElementOfType(target, "span") && target.firstElementChild && isNodeOfType(target.firstElementChild, "ELEMENT_NODE") && isElementOfType(target.firstElementChild, "img")));
	};
	ImageEditPlugin$1.prototype.mouseUpHandler = function(editor, event$1) {
		var selection = editor.getDOMSelection();
		if (selection && selection.type == "image" || this.isEditing) {
			var shouldSelectImage = this.isImageSelection(event$1.rawEvent.target) && event$1.rawEvent.button === MouseRightButton;
			this.applyFormatWithContentModel(editor, this.isCropMode, shouldSelectImage);
		}
	};
	ImageEditPlugin$1.prototype.mouseDownHandler = function(editor, event$1) {
		if (this.isEditing && this.isImageSelection(event$1.rawEvent.target) && event$1.rawEvent.button !== MouseRightButton && !this.isCropMode) this.applyFormatWithContentModel(editor, this.isCropMode);
	};
	ImageEditPlugin$1.prototype.onDropHandler = function(editor) {
		var selection = editor.getDOMSelection();
		if ((selection === null || selection === void 0 ? void 0 : selection.type) == "image") editor.formatContentModel(function(model) {
			var imageDragged = findEditingImage(model, selection.image.id);
			var imageDropped = findEditingImage(model, selection.image.id.replace(DRAG_ID, "").trim());
			if (imageDragged && imageDropped) {
				var draggedIndex = imageDragged.paragraph.segments.indexOf(imageDragged.image);
				mutateBlock(imageDragged.paragraph).segments.splice(draggedIndex, 1);
				var segment = imageDropped.image;
				var paragraph = imageDropped.paragraph;
				mutateSegment(paragraph, segment, function(image) {
					image.isSelected = true;
					image.isSelectedAsImageSelection = true;
				});
				return true;
			}
			return false;
		});
	};
	ImageEditPlugin$1.prototype.keyDownHandler = function(editor, event$1) {
		if (this.isEditing) if (event$1.rawEvent.key === "Escape" || event$1.rawEvent.key === "Delete" || event$1.rawEvent.key === "Backspace") {
			if (event$1.rawEvent.key === "Escape") this.removeImageWrapper();
			this.cleanInfo();
		} else {
			if (event$1.rawEvent.key == "Enter" && this.isCropMode) event$1.rawEvent.preventDefault();
			this.applyFormatWithContentModel(editor, this.isCropMode, true, false);
		}
	};
	ImageEditPlugin$1.prototype.setContentHandler = function(editor) {
		var selection = editor.getDOMSelection();
		if ((selection === null || selection === void 0 ? void 0 : selection.type) == "image") {
			this.cleanInfo();
			setImageState(selection.image, "");
			this.isEditing = false;
			this.isCropMode = false;
		}
	};
	ImageEditPlugin$1.prototype.formatEventHandler = function(event$1) {
		if (this.isEditing && event$1.formatApiName !== IMAGE_EDIT_FORMAT_EVENT) {
			this.cleanInfo();
			this.isEditing = false;
			this.isCropMode = false;
		}
	};
	ImageEditPlugin$1.prototype.contentChangedHandler = function(editor, event$1) {
		switch (event$1.source) {
			case ChangeSource.SetContent:
				this.setContentHandler(editor);
				break;
			case ChangeSource.Format:
				this.formatEventHandler(event$1);
				break;
			case ChangeSource.Drop:
				this.onDropHandler(editor);
				break;
		}
	};
	ImageEditPlugin$1.prototype.applyFormatWithContentModel = function(editor, isCropMode, shouldSelectImage, isApiOperation) {
		var _this = this;
		var editingImageModel;
		var selection = editor.getDOMSelection();
		var isRTL = false;
		editor.formatContentModel(function(model, context) {
			var editingImage = getSelectedImage(model);
			var previousSelectedImage = isApiOperation ? editingImage : findEditingImage(model);
			var result = false;
			context.skipUndoSnapshot = "SkipAll";
			if (shouldSelectImage || (previousSelectedImage === null || previousSelectedImage === void 0 ? void 0 : previousSelectedImage.image) != (editingImage === null || editingImage === void 0 ? void 0 : editingImage.image) || (previousSelectedImage === null || previousSelectedImage === void 0 ? void 0 : previousSelectedImage.image.format.imageState) == "isEditing" || isApiOperation) {
				var _a$5 = _this, lastSrc_1 = _a$5.lastSrc, selectedImage_1 = _a$5.selectedImage, imageEditInfo_1 = _a$5.imageEditInfo, clonedImage_1 = _a$5.clonedImage;
				if ((_this.isEditing || isApiOperation) && previousSelectedImage && lastSrc_1 && selectedImage_1 && imageEditInfo_1 && clonedImage_1) {
					mutateSegment(previousSelectedImage.paragraph, previousSelectedImage.image, function(image) {
						var changeState = applyChange(editor, selectedImage_1, image, imageEditInfo_1, lastSrc_1, _this.wasImageResized || _this.isCropMode, clonedImage_1);
						if (_this.wasImageResized || changeState == "FullyChanged") context.skipUndoSnapshot = false;
						var isSameImage = (previousSelectedImage === null || previousSelectedImage === void 0 ? void 0 : previousSelectedImage.image) === (editingImage === null || editingImage === void 0 ? void 0 : editingImage.image);
						image.isSelected = isSameImage || shouldSelectImage;
						image.isSelectedAsImageSelection = isSameImage || shouldSelectImage;
						image.format.imageState = void 0;
						if ((selection === null || selection === void 0 ? void 0 : selection.type) == "range" && !selection.range.collapsed) {
							if (getSelectedParagraphs(model, true).some(function(paragraph) {
								return paragraph.segments.includes(image);
							})) image.isSelected = true;
						}
					});
					if (shouldSelectImage) normalizeImageSelection(previousSelectedImage);
					_this.cleanInfo();
					result = true;
				}
				_this.isEditing = false;
				_this.isCropMode = false;
				if (editingImage && (selection === null || selection === void 0 ? void 0 : selection.type) == "image" && !shouldSelectImage && !isApiOperation) {
					_this.isEditing = true;
					_this.isCropMode = isCropMode;
					mutateSegment(editingImage.paragraph, editingImage.image, function(image) {
						editingImageModel = image;
						isRTL = editingImage.paragraph.format.direction == "rtl";
						_this.imageEditInfo = updateImageEditInfo(image, selection.image);
						image.format.imageState = "isEditing";
					});
					result = true;
				}
			}
			return result;
		}, {
			onNodeCreated: function(model, node) {
				if (!isApiOperation && editingImageModel && editingImageModel == model && editingImageModel.format.imageState == "isEditing" && isNodeOfType(node, "ELEMENT_NODE") && isElementOfType(node, "img")) if (isCropMode) _this.startCropMode(editor, node, isRTL);
				else _this.startRotateAndResize(editor, node, isRTL);
			},
			apiName: IMAGE_EDIT_FORMAT_EVENT
		}, { tryGetFromCache: true });
	};
	ImageEditPlugin$1.prototype.startEditing = function(editor, image, apiOperation) {
		var _this = this;
		if (!this.imageEditInfo) this.imageEditInfo = getSelectedImageMetadata(editor, image);
		if ((this.imageEditInfo.widthPx == 0 || this.imageEditInfo.heightPx == 0) && !image.complete) {
			image.onload = function() {
				_this.updateImageDimensionsIfZero(image);
				_this.startEditingInternal(editor, image, apiOperation);
				image.onload = null;
				image.onerror = null;
			};
			image.onerror = function() {
				image.onload = null;
				image.onerror = null;
			};
		} else {
			this.updateImageDimensionsIfZero(image);
			this.startEditingInternal(editor, image, apiOperation);
		}
	};
	ImageEditPlugin$1.prototype.updateImageDimensionsIfZero = function(image) {
		var _a$5, _b$1;
		if (((_a$5 = this.imageEditInfo) === null || _a$5 === void 0 ? void 0 : _a$5.widthPx) === 0 || ((_b$1 = this.imageEditInfo) === null || _b$1 === void 0 ? void 0 : _b$1.heightPx) === 0) {
			this.imageEditInfo.widthPx = image.clientWidth;
			this.imageEditInfo.heightPx = image.clientHeight;
		}
	};
	ImageEditPlugin$1.prototype.startEditingInternal = function(editor, image, apiOperation) {
		if (!this.imageEditInfo) this.imageEditInfo = getSelectedImageMetadata(editor, image);
		this.imageHTMLOptions = getHTMLImageOptions(editor, this.options, this.imageEditInfo);
		this.lastSrc = image.getAttribute("src");
		var _a$5 = createImageWrapper(editor, image, this.options, this.imageEditInfo, this.imageHTMLOptions, apiOperation), resizers = _a$5.resizers, rotators = _a$5.rotators, wrapper = _a$5.wrapper, shadowSpan = _a$5.shadowSpan, imageClone = _a$5.imageClone, croppers = _a$5.croppers;
		this.shadowSpan = shadowSpan;
		this.selectedImage = image;
		this.wrapper = wrapper;
		this.clonedImage = imageClone;
		this.wasImageResized = checkIfImageWasResized(image);
		this.resizers = resizers;
		this.rotators = rotators;
		this.croppers = croppers;
		this.zoomScale = editor.getDOMHelper().calculateZoomScale();
		editor.setEditorStyle(IMAGE_EDIT_CLASS, "outline-style:none!important;", ["span:has(>img" + getSafeIdSelector(this.selectedImage.id) + ")"]);
		editor.setEditorStyle(IMAGE_EDIT_CLASS_CARET, "caret-color: transparent;");
	};
	ImageEditPlugin$1.prototype.startRotateAndResize = function(editor, image, isRTL) {
		var _this = this;
		var _a$5, _b$1;
		if (this.imageEditInfo) {
			this.startEditing(editor, image, ["resize", "rotate"]);
			if (this.selectedImage && this.imageEditInfo && this.wrapper && this.clonedImage) {
				var isMobileOrTable = !!editor.getEnvironment().isMobileOrTablet;
				this.dndHelpers = __spreadArray(__spreadArray([], __read(getDropAndDragHelpers(this.wrapper, this.imageEditInfo, this.options, ImageEditElementClass.ResizeHandle, Resizer, function() {
					if (_this.imageEditInfo && _this.selectedImage && _this.wrapper && _this.clonedImage) {
						updateWrapper(_this.imageEditInfo, _this.options, _this.selectedImage, _this.clonedImage, _this.wrapper, _this.resizers, void 0, isRTL);
						_this.wasImageResized = true;
					}
				}, this.zoomScale, isMobileOrTable)), false), __read(getDropAndDragHelpers(this.wrapper, this.imageEditInfo, this.options, ImageEditElementClass.RotateHandle, Rotator, function() {
					var _a$6, _b$2;
					if (_this.imageEditInfo && _this.selectedImage && _this.wrapper && _this.clonedImage) {
						updateWrapper(_this.imageEditInfo, _this.options, _this.selectedImage, _this.clonedImage, _this.wrapper, void 0, void 0, isRTL, true);
						_this.updateRotateHandleState(editor, _this.selectedImage, _this.wrapper, _this.rotators, (_a$6 = _this.imageEditInfo) === null || _a$6 === void 0 ? void 0 : _a$6.angleRad, !!((_b$2 = _this.options) === null || _b$2 === void 0 ? void 0 : _b$2.disableSideResize));
						_this.updateResizeHandleDirection(_this.resizers, _this.imageEditInfo.angleRad);
					}
				}, this.zoomScale, isMobileOrTable)), false);
				updateWrapper(this.imageEditInfo, this.options, this.selectedImage, this.clonedImage, this.wrapper, this.resizers, void 0, isRTL);
				this.updateRotateHandleState(editor, this.selectedImage, this.wrapper, this.rotators, (_a$5 = this.imageEditInfo) === null || _a$5 === void 0 ? void 0 : _a$5.angleRad, !!((_b$1 = this.options) === null || _b$1 === void 0 ? void 0 : _b$1.disableSideResize));
			}
		}
	};
	ImageEditPlugin$1.prototype.updateResizeHandleDirection = function(resizers, angleRad) {
		var resizeHandles = filterInnerResizerHandles(resizers);
		if (angleRad !== void 0) updateHandleCursor(resizeHandles, angleRad);
	};
	ImageEditPlugin$1.prototype.updateRotateHandleState = function(editor, image, wrapper, rotators, angleRad, disableSideResize) {
		var viewport = editor.getVisibleViewport();
		var smallImage = isASmallImage(image.width, image.height);
		if (viewport && rotators && rotators.length > 0) {
			var rotator = rotators[0];
			var rotatorHandle = rotator.firstElementChild;
			if (isNodeOfType(rotatorHandle, "ELEMENT_NODE") && isElementOfType(rotatorHandle, "div")) updateRotateHandle(viewport, angleRad !== null && angleRad !== void 0 ? angleRad : 0, wrapper, rotator, rotatorHandle, smallImage, disableSideResize);
		}
	};
	ImageEditPlugin$1.prototype.isOperationAllowed = function(operation) {
		return operation === "resize" || operation === "rotate" || operation === "flip" || operation === "crop";
	};
	ImageEditPlugin$1.prototype.canRegenerateImage = function(image) {
		return canRegenerateImage(image);
	};
	ImageEditPlugin$1.prototype.startCropMode = function(editor, image, isRTL) {
		var _this = this;
		if (this.imageEditInfo) {
			this.startEditing(editor, image, ["crop"]);
			if (this.imageEditInfo && this.selectedImage && this.wrapper && this.clonedImage) {
				this.dndHelpers = __spreadArray([], __read(getDropAndDragHelpers(this.wrapper, this.imageEditInfo, this.options, ImageEditElementClass.CropHandle, Cropper, function() {
					if (_this.imageEditInfo && _this.selectedImage && _this.wrapper && _this.clonedImage) {
						updateWrapper(_this.imageEditInfo, _this.options, _this.selectedImage, _this.clonedImage, _this.wrapper, void 0, _this.croppers, isRTL);
						_this.isCropMode = true;
					}
				}, this.zoomScale, !!editor.getEnvironment().isMobileOrTablet)), false);
				updateWrapper(this.imageEditInfo, this.options, this.selectedImage, this.clonedImage, this.wrapper, void 0, this.croppers, isRTL);
			}
		}
	};
	ImageEditPlugin$1.prototype.cropImage = function() {
		if (!this.editor) return;
		if (!this.editor.getEnvironment().isSafari) this.editor.focus();
		var selection = this.editor.getDOMSelection();
		if ((selection === null || selection === void 0 ? void 0 : selection.type) == "image") this.applyFormatWithContentModel(this.editor, true, false);
	};
	ImageEditPlugin$1.prototype.editImage = function(editor, image, apiOperation, operation) {
		this.startEditing(editor, image, apiOperation);
		if (!this.selectedImage || !this.imageEditInfo || !this.wrapper || !this.clonedImage) return;
		operation(this.imageEditInfo);
		updateWrapper(this.imageEditInfo, this.options, this.selectedImage, this.clonedImage, this.wrapper);
		this.applyFormatWithContentModel(editor, false, true, true);
	};
	ImageEditPlugin$1.prototype.cleanInfo = function() {
		var _a$5, _b$1;
		(_a$5 = this.editor) === null || _a$5 === void 0 || _a$5.setEditorStyle(IMAGE_EDIT_CLASS, null);
		(_b$1 = this.editor) === null || _b$1 === void 0 || _b$1.setEditorStyle(IMAGE_EDIT_CLASS_CARET, null);
		this.selectedImage = null;
		this.shadowSpan = null;
		this.wrapper = null;
		this.imageEditInfo = null;
		this.imageHTMLOptions = null;
		this.dndHelpers.forEach(function(helper) {
			return helper.dispose();
		});
		this.dndHelpers = [];
		this.clonedImage = null;
		this.lastSrc = null;
		this.wasImageResized = false;
		this.isCropMode = false;
		this.resizers = [];
		this.rotators = [];
		this.croppers = [];
	};
	ImageEditPlugin$1.prototype.removeImageWrapper = function() {
		var image = null;
		if (this.shadowSpan && this.shadowSpan.parentElement) {
			if (this.shadowSpan.firstElementChild && isNodeOfType(this.shadowSpan.firstElementChild, "ELEMENT_NODE") && isElementOfType(this.shadowSpan.firstElementChild, "img")) image = this.shadowSpan.firstElementChild;
			unwrap$1(this.shadowSpan);
			this.shadowSpan = null;
			this.wrapper = null;
		}
		return image;
	};
	ImageEditPlugin$1.prototype.flipImage = function(direction) {
		var _a$5;
		var selection = (_a$5 = this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getDOMSelection();
		if (!this.editor || !selection || selection.type !== "image") return;
		var image = selection.image;
		if (this.editor) this.editImage(this.editor, image, ["flip"], function(imageEditInfo) {
			var angleRad = imageEditInfo.angleRad || 0;
			if (angleRad >= Math.PI / 2 && angleRad < 3 * Math.PI / 4 || angleRad <= -Math.PI / 2 && angleRad > -3 * Math.PI / 4) if (direction === "horizontal") imageEditInfo.flippedVertical = !imageEditInfo.flippedVertical;
			else imageEditInfo.flippedHorizontal = !imageEditInfo.flippedHorizontal;
			else if (direction === "vertical") imageEditInfo.flippedVertical = !imageEditInfo.flippedVertical;
			else imageEditInfo.flippedHorizontal = !imageEditInfo.flippedHorizontal;
		});
	};
	ImageEditPlugin$1.prototype.rotateImage = function(angleRad) {
		var _a$5;
		var selection = (_a$5 = this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getDOMSelection();
		if (!this.editor || !selection || selection.type !== "image") return;
		var image = selection.image;
		if (this.editor) this.editImage(this.editor, image, [], function(imageEditInfo) {
			imageEditInfo.angleRad = (imageEditInfo.angleRad || 0) + angleRad;
		});
	};
	return ImageEditPlugin$1;
})();
function fixupHiddenProperties(editor, options) {
	if (options.undeletableLinkChecker) checkUndeletable(editor, options.undeletableLinkChecker);
}
function checkUndeletable(editor, checker) {
	var e_1, _a$5;
	var anchors = editor.getDOMHelper().queryElements("a");
	try {
		for (var anchors_1 = __values(anchors), anchors_1_1 = anchors_1.next(); !anchors_1_1.done; anchors_1_1 = anchors_1.next()) {
			var a$1 = anchors_1_1.value;
			if (checker(a$1)) setLinkUndeletable(a$1, true);
		}
	} catch (e_1_1) {
		e_1 = { error: e_1_1 };
	} finally {
		try {
			if (anchors_1_1 && !anchors_1_1.done && (_a$5 = anchors_1.return)) _a$5.call(anchors_1);
		} finally {
			if (e_1) throw e_1.error;
		}
	}
}
(function() {
	function HiddenPropertyPlugin$1(option) {
		this.option = option;
		this.editor = null;
	}
	HiddenPropertyPlugin$1.prototype.getName = function() {
		return "HiddenProperty";
	};
	HiddenPropertyPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
	};
	HiddenPropertyPlugin$1.prototype.dispose = function() {
		this.editor = null;
	};
	HiddenPropertyPlugin$1.prototype.onPluginEvent = function(event$1) {
		if (!this.editor) return;
		if (event$1.eventType == "contentChanged" && event$1.source == ChangeSource.SetContent) fixupHiddenProperties(this.editor, this.option);
	};
	return HiddenPropertyPlugin$1;
})();
var MAX_TOUCH_MOVE_DISTANCE = 6;
var POINTER_DETECTION_DELAY = 150;
var PUNCTUATION_MATCHING_REGEX = /[.,;:!]/;
var SPACE_MATCHING_REGEX = /\s/;
(function() {
	function TouchPlugin$1() {
		this.editor = null;
		this.timer = 0;
		this.isDblClicked = false;
		this.isTouchPenPointerEvent = false;
	}
	TouchPlugin$1.prototype.getName = function() {
		return "Touch";
	};
	TouchPlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
		this.isDblClicked = false;
	};
	TouchPlugin$1.prototype.dispose = function() {
		var _a$5, _b$1, _c$1;
		if (this.timer) {
			(_c$1 = (_b$1 = (_a$5 = this.editor) === null || _a$5 === void 0 ? void 0 : _a$5.getDocument()) === null || _b$1 === void 0 ? void 0 : _b$1.defaultView) === null || _c$1 === void 0 || _c$1.clearTimeout(this.timer);
			this.timer = 0;
		}
		this.editor = null;
	};
	TouchPlugin$1.prototype.onPluginEvent = function(event$1) {
		var _this = this;
		var _a$5, _b$1, _c$1, _d$1;
		if (!this.editor) return;
		switch (event$1.eventType) {
			case "pointerDown":
				this.isDblClicked = false;
				this.isTouchPenPointerEvent = true;
				event$1.originalEvent.preventDefault();
				var targetWindow = ((_a$5 = this.editor.getDocument()) === null || _a$5 === void 0 ? void 0 : _a$5.defaultView) || window;
				if (this.timer) targetWindow.clearTimeout(this.timer);
				this.timer = targetWindow.setTimeout(function() {
					_this.timer = 0;
					if (_this.editor) {
						if (!_this.isDblClicked) {
							_this.editor.focus();
							var caretPosition$1 = getNodePositionFromEvent(_this.editor, event$1.rawEvent.x, event$1.rawEvent.y);
							var newRange$1 = _this.editor.getDocument().createRange();
							if (caretPosition$1) {
								var node$1 = caretPosition$1.node, offset$1 = caretPosition$1.offset;
								newRange$1.setStart(node$1, offset$1);
								newRange$1.setEnd(node$1, offset$1);
								var nodeTextContent$1 = node$1.textContent || "";
								var charAtSelection = nodeTextContent$1[offset$1];
								if (node$1.nodeType === Node.TEXT_NODE && charAtSelection && !SPACE_MATCHING_REGEX.test(charAtSelection) && !PUNCTUATION_MATCHING_REGEX.test(charAtSelection)) {
									var _a$6 = findWordBoundaries(nodeTextContent$1, offset$1), wordStart$1 = _a$6.wordStart, wordEnd$1 = _a$6.wordEnd;
									var leftCursorWordLength = offset$1 - wordStart$1;
									var rightCursorWordLength = wordEnd$1 - offset$1;
									var movingOffset = leftCursorWordLength >= rightCursorWordLength ? rightCursorWordLength : -leftCursorWordLength;
									movingOffset = Math.abs(movingOffset) > MAX_TOUCH_MOVE_DISTANCE ? 0 : movingOffset;
									var newOffsetPosition = offset$1 + movingOffset;
									if (movingOffset !== 0 && nodeTextContent$1.length >= newOffsetPosition) {
										newRange$1.setStart(node$1, newOffsetPosition);
										newRange$1.setEnd(node$1, newOffsetPosition);
									}
								}
							}
							_this.editor.setDOMSelection({
								type: "range",
								range: newRange$1,
								isReverted: false
							});
							_this.isTouchPenPointerEvent = false;
						}
					}
				}, POINTER_DETECTION_DELAY);
				break;
			case "doubleClick":
				if (this.isTouchPenPointerEvent) {
					event$1.rawEvent.preventDefault();
					this.isDblClicked = true;
					var caretPosition = getNodePositionFromEvent(this.editor, event$1.rawEvent.x, event$1.rawEvent.y);
					if (caretPosition) {
						var node = caretPosition.node, offset = caretPosition.offset;
						if (node.nodeType !== Node.TEXT_NODE) return;
						var nodeTextContent = node.nodeValue || "";
						var char = nodeTextContent.charAt(offset);
						if (PUNCTUATION_MATCHING_REGEX.test(char)) {
							var newRange = (_b$1 = this.editor.getDocument()) === null || _b$1 === void 0 ? void 0 : _b$1.createRange();
							if (newRange) {
								newRange.setStart(node, offset);
								newRange.setEnd(node, offset + 1);
								this.editor.setDOMSelection({
									type: "range",
									range: newRange,
									isReverted: false
								});
							}
						} else if (SPACE_MATCHING_REGEX.test(char)) {
							var rightSideOfChar = nodeTextContent.substring(offset, nodeTextContent.length);
							if (rightSideOfChar.length > 0 && !/\S/.test(rightSideOfChar)) {
								var start = offset;
								while (start > 0 && SPACE_MATCHING_REGEX.test(nodeTextContent.charAt(start - 1))) start--;
								var newRange = (_c$1 = this.editor.getDocument()) === null || _c$1 === void 0 ? void 0 : _c$1.createRange();
								if (newRange) {
									newRange.setStart(node, start);
									newRange.setEnd(node, start + 1);
									this.editor.setDOMSelection({
										type: "range",
										range: newRange,
										isReverted: false
									});
								}
							}
						} else {
							var _e$1 = findWordBoundaries(nodeTextContent, offset), wordStart = _e$1.wordStart, wordEnd = _e$1.wordEnd;
							var newRange = (_d$1 = this.editor.getDocument()) === null || _d$1 === void 0 ? void 0 : _d$1.createRange();
							if (newRange) {
								newRange.setStart(node, wordStart);
								newRange.setEnd(node, wordEnd);
								this.editor.setDOMSelection({
									type: "range",
									range: newRange,
									isReverted: false
								});
							}
						}
					}
				}
				break;
		}
	};
	return TouchPlugin$1;
})();
function findWordBoundaries(text, offset) {
	var start = offset;
	var end = offset;
	while (start > 0 && !SPACE_MATCHING_REGEX.test(text[start - 1]) && !PUNCTUATION_MATCHING_REGEX.test(text[start - 1])) start--;
	while (end < text.length && !SPACE_MATCHING_REGEX.test(text[end]) && !PUNCTUATION_MATCHING_REGEX.test(text[end])) end++;
	return {
		wordStart: start,
		wordEnd: end
	};
}
function setMarkedIndex(editor, context, index$1, alternativeRange) {
	context.replaceHighlight.clear();
	context.markedIndex = index$1;
	var range = context.ranges[context.markedIndex];
	if (range) {
		context.replaceHighlight.addRanges([range]);
		var rect = void 0;
		if (context.scrollMargin >= 0 && (rect = editor.getVisibleViewport())) scrollRectIntoView(editor.getScrollContainer(), rect, editor.getDOMHelper(), range.getBoundingClientRect(), context.scrollMargin, true);
	} else context.markedIndex = -1;
	editor.triggerEvent("findResultChanged", {
		markedIndex: context.markedIndex,
		ranges: context.ranges,
		alternativeRange
	});
}
function sortRanges(ranges) {
	return ranges.sort(compareRange);
}
function compareRange(r1, r2) {
	if (r1.startContainer == r2.startContainer) return r1.startOffset - r2.startOffset;
	else return r1.startContainer.compareDocumentPosition(r2.startContainer) & Node.DOCUMENT_POSITION_FOLLOWING ? -1 : 1;
}
function updateHighlight(editor, context, addedBlockElements, removedBlockElements) {
	var _a$5;
	if (addedBlockElements === void 0) addedBlockElements = null;
	if (removedBlockElements === void 0) removedBlockElements = null;
	context.findHighlight.clear();
	if (context.text) {
		var text_1 = context.text, matchCase_1 = context.matchCase, wholeWord_1 = context.wholeWord;
		var domHelper_1 = editor.getDOMHelper();
		if (removedBlockElements) context.ranges = context.ranges.filter(function(r$1) {
			return !removedBlockElements.some(function(x$1) {
				return x$1.contains(r$1.startContainer);
			}) && domHelper_1.isNodeInEditor(r$1.startContainer, true);
		});
		else context.ranges = [];
		if (addedBlockElements) {
			var newRanges = addedBlockElements.map(function(b$1) {
				return getRangesByText(b$1, text_1, matchCase_1, wholeWord_1, true);
			});
			context.ranges = (_a$5 = context.ranges).concat.apply(_a$5, __spreadArray([], __read(newRanges), false));
		} else context.ranges = domHelper_1.getRangesByText(text_1, matchCase_1, wholeWord_1);
		sortRanges(context.ranges);
	} else context.ranges = [];
	if (context.ranges.length > 0) context.findHighlight.addRanges(context.ranges);
	setMarkedIndex(editor, context, -1);
}
var FindHighlightRuleKey = "_RoosterjsFindHighlight";
var FindHighlightSelector = "::highlight(roostersFindHighlight)";
var ReplaceHighlightRuleKey = "_RoosterjsReplaceHighlight";
var ReplaceHighlightSelector = "::highlight(roostersReplaceHighlight)";
var DefaultFindHighlightStyle = "background-color: yellow;";
var DefaultReplaceHighlightStyle = "background-color: orange;";
(function() {
	function FindReplacePlugin$1(context, options) {
		var _a$5, _b$1;
		this.context = context;
		this.editor = null;
		this.findHighlightStyle = (_a$5 = options === null || options === void 0 ? void 0 : options.findHighlightStyle) !== null && _a$5 !== void 0 ? _a$5 : DefaultFindHighlightStyle;
		this.replaceHighlightStyle = (_b$1 = options === null || options === void 0 ? void 0 : options.replaceHighlightStyle) !== null && _b$1 !== void 0 ? _b$1 : DefaultReplaceHighlightStyle;
	}
	FindReplacePlugin$1.prototype.getName = function() {
		return "FindReplace";
	};
	FindReplacePlugin$1.prototype.initialize = function(editor) {
		var _a$5;
		this.editor = editor;
		var win = (_a$5 = editor.getDocument().defaultView) !== null && _a$5 !== void 0 ? _a$5 : window;
		this.context.findHighlight.initialize(win);
		this.context.replaceHighlight.initialize(win);
		this.editor.setEditorStyle(FindHighlightRuleKey, this.findHighlightStyle, [FindHighlightSelector]);
		this.editor.setEditorStyle(ReplaceHighlightRuleKey, this.replaceHighlightStyle, [ReplaceHighlightSelector]);
	};
	FindReplacePlugin$1.prototype.dispose = function() {
		this.context.findHighlight.dispose();
		this.context.replaceHighlight.dispose();
		if (this.editor) {
			this.editor.setEditorStyle(FindHighlightRuleKey, null);
			this.editor.setEditorStyle(ReplaceHighlightRuleKey, null);
			this.editor = null;
		}
	};
	FindReplacePlugin$1.prototype.onPluginEvent = function(event$1) {
		if (!this.context.text || !this.editor) return;
		switch (event$1.eventType) {
			case "input":
				var selection = this.editor.getDOMSelection();
				if ((selection === null || selection === void 0 ? void 0 : selection.type) == "range") {
					var block = this.editor.getDOMHelper().findClosestBlockElement(selection.range.startContainer);
					updateHighlight(this.editor, this.context, [block], [block]);
				}
				break;
			case "contentChanged":
				if (!event$1.contentModel && event$1.source != ChangeSource.Replace) updateHighlight(this.editor, this.context);
				break;
			case "rewriteFromModel":
				updateHighlight(this.editor, this.context, event$1.addedBlockElements, event$1.removedBlockElements);
				break;
		}
	};
	return FindReplacePlugin$1;
})();
function isWindowWithHighlight(win) {
	return typeof win.Highlight === "function" && typeof win.CSS === "object";
}
function isHighlightRegistryWithMap(highlight) {
	return typeof highlight.set === "function";
}
(function() {
	function HighlightHelperImpl$1(styleKey) {
		this.styleKey = styleKey;
	}
	HighlightHelperImpl$1.prototype.initialize = function(win) {
		if (isWindowWithHighlight(win) && isHighlightRegistryWithMap(win.CSS.highlights)) {
			this.highlights = win.CSS.highlights;
			this.highlight = new win.Highlight();
			this.highlights.set(this.styleKey, this.highlight);
		}
	};
	HighlightHelperImpl$1.prototype.dispose = function() {
		var _a$5, _b$1;
		(_a$5 = this.highlights) === null || _a$5 === void 0 || _a$5.delete(this.styleKey);
		(_b$1 = this.highlight) === null || _b$1 === void 0 || _b$1.clear();
		this.highlight = void 0;
		this.highlights = void 0;
	};
	HighlightHelperImpl$1.prototype.addRanges = function(ranges) {
		var e_1, _a$5;
		if (this.highlight) try {
			for (var ranges_1 = __values(ranges), ranges_1_1 = ranges_1.next(); !ranges_1_1.done; ranges_1_1 = ranges_1.next()) {
				var range = ranges_1_1.value;
				this.highlight.add(range);
			}
		} catch (e_1_1) {
			e_1 = { error: e_1_1 };
		} finally {
			try {
				if (ranges_1_1 && !ranges_1_1.done && (_a$5 = ranges_1.return)) _a$5.call(ranges_1);
			} finally {
				if (e_1) throw e_1.error;
			}
		}
	};
	HighlightHelperImpl$1.prototype.clear = function() {
		var _a$5;
		(_a$5 = this.highlight) === null || _a$5 === void 0 || _a$5.clear();
	};
	return HighlightHelperImpl$1;
})();
function retrieveStringFromParsedTable(tsInfo) {
	var parsedTable = tsInfo.parsedTable, firstCo = tsInfo.firstCo, lastCo = tsInfo.lastCo;
	var result = "";
	if (lastCo) for (var r$1 = firstCo.row; r$1 <= lastCo.row; r$1++) for (var c$1 = firstCo.col; c$1 <= lastCo.col; c$1++) {
		var cell = parsedTable[r$1] && parsedTable[r$1][c$1];
		if (cell && typeof cell != "string") result += " " + cell.innerText + ",";
	}
	return result;
}
function getIsSelectingOrUnselecting(prevTableSelection, newTableSelection) {
	if (!prevTableSelection) return "selecting";
	var prevFirstRow = prevTableSelection.firstRow, prevLastRow = prevTableSelection.lastRow, prevFirstColumn = prevTableSelection.firstColumn, prevLastColumn = prevTableSelection.lastColumn;
	var newFirstRow = newTableSelection.firstRow, newLastRow = newTableSelection.lastRow, newFirstColumn = newTableSelection.firstColumn, newLastColumn = newTableSelection.lastColumn;
	var prevArea = (Math.abs(prevLastRow - prevFirstRow) + 1) * (Math.abs(prevLastColumn - prevFirstColumn) + 1);
	var newArea = (Math.abs(newLastRow - newFirstRow) + 1) * (Math.abs(newLastColumn - newFirstColumn) + 1);
	if (prevFirstRow === newFirstRow && prevLastRow === newLastRow && prevFirstColumn === newFirstColumn && prevLastColumn === newLastColumn) return null;
	if (newArea > prevArea) return "selecting";
	else if (newArea < prevArea) return "unselecting";
	else return "selecting";
	if (prevFirstColumn !== newFirstColumn || prevFirstRow !== newFirstRow || prevLastColumn !== newLastColumn || prevLastRow !== newLastRow) return "selecting";
	return null;
}
(function() {
	function AnnouncePlugin$1() {
		this.editor = null;
		this.previousSelection = null;
	}
	AnnouncePlugin$1.prototype.getName = function() {
		return "Announce";
	};
	AnnouncePlugin$1.prototype.initialize = function(editor) {
		this.editor = editor;
	};
	AnnouncePlugin$1.prototype.dispose = function() {
		this.editor = null;
		this.previousSelection = null;
	};
	AnnouncePlugin$1.prototype.onPluginEvent = function(event$1) {
		var _a$5, _b$1;
		if (!this.editor) return;
		if (event$1.eventType == "selectionChanged") {
			if (((_a$5 = event$1.newSelection) === null || _a$5 === void 0 ? void 0 : _a$5.type) == "table") {
				var action = getIsSelectingOrUnselecting(((_b$1 = this.previousSelection) === null || _b$1 === void 0 ? void 0 : _b$1.type) == "table" ? this.previousSelection : null, event$1.newSelection);
				if (action && event$1.newSelection.tableSelectionInfo) this.editor.announce({
					defaultStrings: action === "unselecting" ? "unselected" : "selected",
					formatStrings: [retrieveStringFromParsedTable(event$1.newSelection.tableSelectionInfo)]
				});
			}
			this.previousSelection = event$1.newSelection;
		}
	};
	return AnnouncePlugin$1;
})();
function createEditor(contentDiv, additionalPlugins, initialModel) {
	return new Editor(contentDiv, {
		plugins: __spreadArray([
			new PastePlugin(),
			new EditPlugin(),
			new ShortcutPlugin()
		], __read(additionalPlugins !== null && additionalPlugins !== void 0 ? additionalPlugins : []), false),
		initialModel,
		defaultSegmentFormat: {
			fontFamily: "Calibri,Arial,Helvetica,sans-serif",
			fontSize: "11pt",
			textColor: "#000000"
		}
	});
}
var HeaderFontSizes = {
	h1: "2em",
	h2: "1.5em",
	h3: "1.17em",
	h4: "1em",
	h5: "0.83em",
	h6: "0.67em"
};
HeaderFontSizes.h1, HeaderFontSizes.h2, HeaderFontSizes.h3, HeaderFontSizes.h4, HeaderFontSizes.h5, HeaderFontSizes.h6;
var root_2$3 = from_html(`<button type="button"><!></button>`);
var root_5$2 = from_html(`<button type="button"> </button>`);
var root_6$2 = from_html(`<button type="button"> </button>`);
var root_4$3 = from_html(`<div class="aME"><div class="flex items-start flex-col md:flex-row md:items-center aME"><div class="w-80 text-sm font-semibold text-slate-600 mb-8 aME">Columnas</div> <div class="flex flex-wrap items-center aME"></div></div> <div class="flex items-start flex-col md:flex-row md:items-center aME"><div class="w-80 text-sm font-semibold text-slate-600 mb-8 aME">Filas</div> <div class="flex flex-wrap items-center aME"></div></div></div>`);
var root_8$1 = from_html(`<div class="h-20 w-32 m-4 border border-black/60 cursor-pointer hover:outline hover:outline-1 hover:outline-black/70 aME"></div>`);
var root_7$2 = from_html(`<div class="flex flex-wrap w-full aME"></div>`);
var root_10 = from_html(`<button type="button" class="px-12 py-4 hover:bg-slate-100 rounded border border-slate-200"> </button>`);
var root_9 = from_html(`<div class="flex flex-wrap w-full gap-8 aME"></div>`);
var root_3$3 = from_html(`<div class="absolute top-[47px] w-[calc(100%-20px)] left-[10px] border border-[#ab9efc] min-h-[60px] bg-white rounded-lg z-50 shadow-lg p-12 _10 aME" role="dialog" aria-label="Editor popup" tabindex="-1"><input type="text" inputmode="none" autocomplete="off" class="opacity-0 h-2 w-1 absolute z-[-1]"/> <!> <!> <!></div>`);
var root_12 = from_html(`<button type="button" class="_4 aME"> </button>`);
var root_1$6 = from_html(`<div class="flex flex-wrap gap-2 items-center p-6 border border-slate-200 rounded-t-[6px] bg-slate-50 relative aME" role="toolbar" aria-label="Editor toolbar" tabindex="-1"><!> <!> <!></div> <div class="border border-slate-200 rounded-b-[6px] bg-white min-h-[14rem] shadow-inner aME" style="position: relative;"><div class="aL3 min-h-[14rem] p-16 text-base leading-relaxed text-slate-900 outline-none empty:before:content-[attr(data-placeholder)] empty:before:text-slate-400 empty:before:pointer-events-none aME" role="textbox" aria-label="Rich text editor" aria-multiline="true" contenteditable="true" data-placeholder="Empieza a escribir contenido enriquecido…"></div></div>`, 1);
var root_13 = from_html(`<div class="border-2 border-dashed border-slate-300 rounded-xl p-16 text-center text-slate-600 bg-slate-50 aME">El editor de texto enriquecido se cargará cuando estés en el navegador.</div>`);
var root$11 = from_html(`<div><!></div>`);
function HTMLEditor($$anchor, $$props) {
	push($$props, true);
	const saveOn = prop($$props, "saveOn", 7);
	const getInitialValue = () => saveOn() && $$props.save && (saveOn()[$$props.save] ?? "") || "";
	let editorRoot = state(null);
	let editorContainer = state(null);
	let editor = state(null);
	let editorValue = state(proxy(getInitialValue()));
	let formatState = state(proxy({}));
	let isInTable = state(false);
	let fontSizeOptions = state(proxy([
		"8pt",
		"9pt",
		"10pt",
		"11pt",
		"12pt",
		"14pt",
		"16pt",
		"18pt",
		"20pt",
		"24pt",
		"28pt",
		"32pt"
	]));
	let fontSizeSelection = state("16pt");
	let headingSelection = state(0);
	let textColorValue = state("#000000");
	let backgroundColorValue = state("#ffffff");
	let fontFamilySelection = state("Arial");
	let lastSyncedValue = getInitialValue();
	let pendingFormatFrame = null;
	let showTableLayer = state(false);
	let showTextColorLayer = state(false);
	let showBackgroudColorLayer = state(false);
	let showTextSizeLayer = state(false);
	let selectedRows = state(0);
	let selectedCols = state(0);
	const getEditorCore = () => {
		return get(editor)?.core;
	};
	const ensureFontSizeOption = (size) => {
		if (!size) return;
		if (!get(fontSizeOptions).includes(size)) set(fontSizeOptions, [...get(fontSizeOptions), size], true);
	};
	const createModelFromHtml = (html$1, core) => {
		if (!browser) return void 0;
		const normalized = html$1?.trim() ? html$1 : "<p></p>";
		const body = new DOMParser().parseFromString(normalized, "text/html").body;
		if (!body) return void 0;
		return domToContentModel(body, core ? createDomToModelContextWithConfig(core.environment.domToModelSettings.calculated, core.api.createEditorContext(core, false)) : createDomToModelContext());
	};
	let layerInput = state(void 0);
	let avoidCloseOnBlur = false;
	const sanitizeHtml = (html$1) => {
		if (!browser) return html$1;
		const doc = new DOMParser().parseFromString(html$1, "text/html");
		const removeStyles = (elements, props) => {
			elements.forEach((el) => props.forEach((p$1) => el.style.removeProperty(p$1)));
		};
		removeStyles(doc.querySelectorAll("div"), [
			"margin-top",
			"margin-bottom",
			"font-family"
		]);
		removeStyles(doc.querySelectorAll("td, th"), [
			"width",
			"height",
			"border-width",
			"border-style",
			"border-color"
		]);
		return doc.body.innerHTML;
	};
	const applyHtmlToEditor = (html$1) => {
		const core = getEditorCore();
		if (!core) return;
		const model = createModelFromHtml(sanitizeHtml(html$1), core);
		if (!model) return;
		core.api.setContentModel(core, model, { ignoreSelection: true });
		core.api.triggerEvent(core, {
			eventType: "contentChanged",
			source: ChangeSource.SetContent
		}, true);
		setTimeout(() => {
			if (get(editorRoot)) removeUnwantedStyles(get(editorRoot));
		}, 0);
	};
	const removeUnwantedStyles = (element) => {
		const removeStyles = (elements, props) => {
			elements.forEach((el) => props.forEach((p$1) => el.style.removeProperty(p$1)));
		};
		removeStyles(element.querySelectorAll("div"), [
			"margin-top",
			"margin-bottom",
			"font-family",
			"font-size",
			"color"
		]);
		removeStyles(element.querySelectorAll("td, th"), [
			"width",
			"height",
			"border-width",
			"border-style",
			"border-color"
		]);
	};
	const syncEditorHtml = () => {
		if (!get(editorRoot)) return;
		removeUnwantedStyles(get(editorRoot));
		const nextValue = get(editorRoot).innerHTML;
		if (nextValue !== get(editorValue)) set(editorValue, nextValue, true);
	};
	const checkIfInTable = () => {
		if (!get(editor) || !get(editorRoot)) {
			set(isInTable, false);
			return;
		}
		const selection = get(editor).getDOMSelection();
		if (!selection) {
			set(isInTable, false);
			return;
		}
		if (selection.type === "table") {
			set(isInTable, true);
			return;
		}
		if (selection.type === "range") {
			const range = selection.range;
			const nodesToCheck = [range.startContainer, range.commonAncestorContainer];
			for (const startNode of nodesToCheck) {
				let node = startNode;
				while (node && node !== get(editorRoot)) {
					if (node.nodeType === Node.ELEMENT_NODE) {
						const tagName = node.tagName?.toUpperCase();
						if ([
							"TABLE",
							"TD",
							"TH",
							"TR",
							"TBODY",
							"THEAD",
							"TFOOT"
						].includes(tagName)) {
							set(isInTable, true);
							return;
						}
					}
					node = node.parentNode;
				}
			}
		}
		set(isInTable, false);
	};
	const refreshFormatState = () => {
		if (!get(editor)) {
			set(formatState, {}, true);
			set(isInTable, false);
			return;
		}
		set(formatState, getFormatState(get(editor)) ?? {}, true);
		if (get(formatState).fontSize) {
			ensureFontSizeOption(get(formatState).fontSize);
			set(fontSizeSelection, get(formatState).fontSize, true);
		}
		set(headingSelection, get(formatState).headingLevel ?? 0, true);
		if (get(formatState).fontName) set(fontFamilySelection, get(formatState).fontName, true);
		checkIfInTable();
	};
	const scheduleFormatStateRefresh = () => {
		if (!browser) {
			refreshFormatState();
			return;
		}
		if (pendingFormatFrame !== null) cancelAnimationFrame(pendingFormatFrame);
		pendingFormatFrame = requestAnimationFrame(() => {
			pendingFormatFrame = null;
			refreshFormatState();
		});
	};
	const createSyncPlugin = () => {
		let pluginEditor = null;
		let mutationObserver = null;
		return {
			getName: () => "svelte-sync",
			initialize: (ed) => {
				pluginEditor = ed;
				set(editor, ed, true);
				syncEditorHtml();
				refreshFormatState();
				if (get(editorRoot) && browser) {
					mutationObserver = new MutationObserver((mutations) => {
						mutations.forEach((mutation) => {
							mutation.addedNodes.forEach((node) => {
								if (node.nodeType === Node.ELEMENT_NODE) {
									const element = node;
									removeUnwantedStyles(element);
									if (element.tagName === "DIV") {
										element.style.removeProperty("margin-top");
										element.style.removeProperty("margin-bottom");
										element.style.removeProperty("font-family");
									} else if (element.tagName === "TD" || element.tagName === "TH") {
										element.style.removeProperty("width");
										element.style.removeProperty("height");
										element.style.removeProperty("border-width");
										element.style.removeProperty("border-style");
										element.style.removeProperty("border-color");
									}
								}
							});
						});
					});
					mutationObserver.observe(get(editorRoot), {
						childList: true,
						subtree: true,
						attributes: true,
						attributeFilter: ["style"]
					});
				}
			},
			dispose: () => {
				if (mutationObserver) {
					mutationObserver.disconnect();
					mutationObserver = null;
				}
				if (pluginEditor && get(editor) === pluginEditor) set(editor, null);
				pluginEditor = null;
			},
			onPluginEvent: (event$1) => {
				if (event$1.eventType === "contentChanged") {
					syncEditorHtml();
					if (get(editorRoot)) setTimeout(() => removeUnwantedStyles(get(editorRoot)), 0);
				}
				if (event$1.eventType === "contentChanged" || event$1.eventType === "selectionChanged") {
					if (event$1.eventType === "selectionChanged") checkIfInTable();
					scheduleFormatStateRefresh();
				}
			}
		};
	};
	const withEditor = (cb) => {
		if (!get(editor)) return;
		get(editor).focus();
		cb(get(editor));
		scheduleFormatStateRefresh();
	};
	const handleAlignment = (alignment) => {
		withEditor((instance) => setAlignment(instance, alignment));
	};
	const handleFontSizeChange = (size) => {
		set(fontSizeSelection, size, true);
		withEditor((instance) => setFontSize(instance, size));
	};
	const handleInsertTable = () => {
		if (get(selectedRows) > 0 && get(selectedCols) > 0) {
			withEditor((instance) => insertTable(instance, get(selectedCols), get(selectedRows)));
			set(showTableLayer, false);
			set(selectedRows, 0);
			set(selectedCols, 0);
		}
	};
	const handleRowSelect = (rows) => {
		set(selectedRows, rows, true);
		if (get(selectedRows) > 0 && get(selectedCols) > 0) handleInsertTable();
	};
	const handleColSelect = (cols) => {
		set(selectedCols, cols, true);
		if (get(selectedRows) > 0 && get(selectedCols) > 0) handleInsertTable();
	};
	const closeTableDialog = () => {
		set(showTableLayer, false);
		set(selectedRows, 0);
		set(selectedCols, 0);
	};
	const handleInsertHR = () => {
		if (get(editor)) {
			const core = getEditorCore();
			if (core) core.api.insertNode(core, document.createElement("hr"));
		}
	};
	const handleTextColorChange = (color) => {
		set(textColorValue, color, true);
		withEditor((instance) => setTextColor(instance, color));
	};
	const handleBackgroundColorChange = (color) => {
		set(backgroundColorValue, color, true);
		withEditor((instance) => setBackgroundColor(instance, color));
	};
	onMount(() => {
		if (!browser || !get(editorRoot)) return;
		const initialModel = createModelFromHtml(get(editorValue));
		const syncPlugin = createSyncPlugin();
		const tableEditPlugin = new TableEditPlugin();
		const instance = createEditor(get(editorRoot), [syncPlugin, tableEditPlugin], initialModel);
		set(editor, instance, true);
		const handleClick = () => setTimeout(checkIfInTable, 0);
		get(editorRoot).addEventListener("click", handleClick);
		get(editorRoot).addEventListener("focus", handleClick);
		return () => {
			if (pendingFormatFrame !== null) {
				cancelAnimationFrame(pendingFormatFrame);
				pendingFormatFrame = null;
			}
			get(editorRoot)?.removeEventListener("click", handleClick);
			get(editorRoot)?.removeEventListener("focus", handleClick);
			instance.dispose();
			set(editor, null);
		};
	});
	const showLayer = user_derived(() => get(showTableLayer) || get(showTextColorLayer) || get(showBackgroudColorLayer) || get(showTextSizeLayer));
	user_effect(() => {
		if (get(showLayer)) get(layerInput)?.focus();
	});
	user_effect(() => {
		if (!saveOn() || !$$props.save) return;
		if (get(editorValue) !== lastSyncedValue) {
			lastSyncedValue = get(editorValue);
			saveOn()[$$props.save] = get(editorValue);
		}
	});
	user_effect(() => {
		if (!browser || !saveOn() || !$$props.save) return;
		const incoming = (saveOn()[$$props.save] ?? "") || "";
		if (incoming !== get(editorValue)) {
			lastSyncedValue = incoming;
			set(editorValue, incoming, true);
			if (get(editor)) applyHtmlToEditor(incoming);
		}
	});
	const toolbarItems = user_derived(() => [
		{
			label: "Bold",
			icon: "<strong>B</strong>",
			action: () => withEditor(toggleBold),
			active: !!get(formatState).isBold
		},
		{
			label: "Italic",
			icon: "<em>I</em>",
			action: () => withEditor(toggleItalic),
			active: !!get(formatState).isItalic
		},
		{
			label: "Text Size",
			icon: `<img class="h-24 w-24" src="${parseSVG(TextSizeIcon)}" alt="" />`,
			action: () => {
				set(showTextSizeLayer, !get(showTextSizeLayer));
			},
			active: get(showTextSizeLayer),
			isLayer: true
		},
		{
			label: "Insert table",
			icon: "⊞",
			action: () => {
				set(showTableLayer, !get(showTableLayer));
			},
			active: get(showTableLayer),
			isLayer: true
		},
		{
			label: "Text color",
			html: `<img class="h-20 w-20 ml-4" src="${parseSVG(TextColorIcon)}" alt="" />
             <div class="absolute bottom-2 left-2 w-[calc(100%-4px)] h-12 border border-black/70 rounded-[2px]" style="background-color: ${get(textColorValue)};"></div>`,
			action: () => {
				set(showTextColorLayer, !get(showTextColorLayer));
			},
			active: get(showTextColorLayer),
			isLayer: true,
			className: "pb-12"
		},
		{
			label: "Background color",
			html: `<img class="h-20 w-20" src="${parseSVG(TextBackgroudColor)}" alt="" />
             <div class="absolute bottom-2 left-2 w-[calc(100%-4px)] h-12 border border-black/70 rounded-[2px]" style="background-color: ${get(backgroundColorValue)};"></div>`,
			action: () => {
				set(showBackgroudColorLayer, !get(showBackgroudColorLayer));
			},
			active: get(showBackgroudColorLayer),
			isLayer: true,
			className: "pb-12"
		},
		{
			label: "Align left",
			icon: "⬅",
			action: () => handleAlignment("left"),
			active: get(formatState).textAlign === "left" || !get(formatState).textAlign
		},
		{
			label: "Align center",
			icon: "⬍",
			action: () => handleAlignment("center"),
			active: get(formatState).textAlign === "center"
		},
		{
			label: "Align right",
			icon: "➡",
			action: () => handleAlignment("right"),
			active: get(formatState).textAlign === "right"
		},
		{
			label: "Underline",
			icon: "<u>U</u>",
			action: () => withEditor(toggleUnderline),
			active: !!get(formatState).isUnderline
		},
		{
			label: "Strikethrough",
			icon: "<s>S</s>",
			action: () => withEditor(toggleStrikethrough),
			active: !!get(formatState).isStrikeThrough
		},
		{
			label: "Justify",
			icon: "☰",
			action: () => handleAlignment("justify"),
			active: get(formatState).textAlign === "justify"
		},
		{
			label: "Bulleted list",
			icon: "••",
			action: () => withEditor(toggleBullet),
			active: !!get(formatState).isBullet
		},
		{
			label: "Numbered list",
			icon: "1.",
			action: () => withEditor(toggleNumbering),
			active: !!get(formatState).isNumbering
		},
		{
			label: "Insert horizontal rule",
			icon: "─",
			action: handleInsertHR
		},
		{
			label: "Clear format",
			icon: "Clear",
			action: () => withEditor(clearFormat)
		},
		{
			label: "Undo",
			icon: "↶",
			action: () => {
				if (get(editor) && getEditorCore()) {
					getEditorCore()?.api.undo(getEditorCore());
					scheduleFormatStateRefresh();
				}
			}
		},
		{
			label: "Redo",
			icon: "↷",
			action: () => {
				if (get(editor) && getEditorCore()) {
					getEditorCore()?.api.redo(getEditorCore());
					scheduleFormatStateRefresh();
				}
			}
		}
	]);
	const tableButtons = [
		{
			label: "Add row above",
			icon: "⬆ Row",
			action: () => withEditor((instance) => editTable(instance, "insertAbove"))
		},
		{
			label: "Add row below",
			icon: "⬇ Row",
			action: () => withEditor((instance) => editTable(instance, "insertBelow"))
		},
		{
			label: "Add column left",
			icon: "⬅ Col",
			action: () => withEditor((instance) => editTable(instance, "insertLeft"))
		},
		{
			label: "Add column right",
			icon: "➡ Col",
			action: () => withEditor((instance) => editTable(instance, "insertRight"))
		},
		{
			label: "Delete row",
			icon: "🗑 Row",
			action: () => withEditor((instance) => editTable(instance, "deleteRow"))
		},
		{
			label: "Delete column",
			icon: "🗑 Col",
			action: () => withEditor((instance) => editTable(instance, "deleteColumn"))
		}
	];
	var div = root$11();
	var node_1 = child(div);
	var consequent_5 = ($$anchor$1) => {
		var fragment = root_1$6();
		var div_1 = first_child(fragment);
		div_1.__click = (e$1) => {
			if (get(showTableLayer) && !e$1.target.closest("._10")) {}
		};
		div_1.__keydown = (e$1) => {
			if (e$1.key === "Escape" && get(showTableLayer)) {}
		};
		var node_2 = child(div_1);
		each(node_2, 17, () => get(toolbarItems), index, ($$anchor$2, item) => {
			var button = root_2$3();
			let classes;
			button.__click = function(...$$args) {
				get(item).action?.apply(this, $$args);
			};
			html(child(button), () => get(item).html || get(item).icon);
			reset(button);
			template_effect(() => {
				button.disabled = !get(editor);
				classes = set_class(button, 1, `_4 ${get(item).active ? "bg-blue-100 border-blue-500 text-blue-700" : ""} ${get(item).className ?? "" ?? ""}`, "aME", classes, { _6: get(item).isLayer && get(item).active });
				set_attribute(button, "aria-label", get(item).label);
			});
			append($$anchor$2, button);
		});
		var node_4 = sibling(node_2, 2);
		var consequent_3 = ($$anchor$2) => {
			var div_2 = root_3$3();
			div_2.__mousedown = () => avoidCloseOnBlur = true;
			div_2.__click = (e$1) => e$1.stopPropagation();
			div_2.__keydown = (e$1) => {
				if (e$1.key === "Escape") closeTableDialog();
			};
			var input = child(div_2);
			bind_this(input, ($$value) => set(layerInput, $$value), () => get(layerInput));
			var node_5 = sibling(input, 2);
			var consequent = ($$anchor$3) => {
				var div_3 = root_4$3();
				var div_4 = child(div_3);
				var div_5 = sibling(child(div_4), 2);
				each(div_5, 20, () => Array(12), index, ($$anchor$4, _, i$1) => {
					const num = user_derived(() => i$1 + 1);
					var button_1 = root_5$2();
					let classes_1;
					button_1.__click = () => handleColSelect(get(num));
					var text = child(button_1, true);
					reset(button_1);
					template_effect(() => {
						classes_1 = set_class(button_1, 1, `mr-4 mb-4 w-32 h-28 flex items-center justify-center rounded bg-indigo-100/50 cursor-pointer text-[13px] text-slate-900 transition-all hover:not(.selected):bg-indigo-100 hover:not(.selected):border-indigo-500 hover:not(.selected):text-indigo-700 ${get(selectedCols) >= get(num) ? "bg-indigo-500 text-white font-semibold" : ""}`, "aME", classes_1, { selected: get(selectedCols) >= get(num) });
						set_attribute(button_1, "aria-label", `Select ${get(num) ?? ""} columns`);
						set_text(text, get(num));
					});
					append($$anchor$4, button_1);
				});
				reset(div_5);
				reset(div_4);
				var div_6 = sibling(div_4, 2);
				var div_7 = sibling(child(div_6), 2);
				each(div_7, 20, () => Array(12), index, ($$anchor$4, _, i$1) => {
					const num = user_derived(() => i$1 + 1);
					var button_2 = root_6$2();
					let classes_2;
					button_2.__click = () => handleRowSelect(get(num));
					var text_1 = child(button_2, true);
					reset(button_2);
					template_effect(() => {
						classes_2 = set_class(button_2, 1, `mr-4 mb-4 w-32 h-28 flex items-center justify-center rounded bg-indigo-100/50 cursor-pointer text-[13px] text-slate-900 transition-all hover:not(.selected):bg-indigo-100 hover:not(.selected):border-indigo-500 hover:not(.selected):text-indigo-700 ${get(selectedRows) >= get(num) ? "bg-indigo-500 text-white font-semibold" : ""}`, "aME", classes_2, { selected: get(selectedRows) >= get(num) });
						set_attribute(button_2, "aria-label", `Select ${get(num) ?? ""} rows`);
						set_text(text_1, get(num));
					});
					append($$anchor$4, button_2);
				});
				reset(div_7);
				reset(div_6);
				reset(div_3);
				append($$anchor$3, div_3);
			};
			if_block(node_5, ($$render) => {
				if (get(showTableLayer)) $$render(consequent);
			});
			var node_6 = sibling(node_5, 2);
			var consequent_1 = ($$anchor$3) => {
				const colors = user_derived(() => get(showBackgroudColorLayer) ? editorBackgroundColors : editorTextColors);
				var div_8 = root_7$2();
				each(div_8, 21, () => get(colors), index, ($$anchor$4, color) => {
					var div_9 = root_8$1();
					div_9.__click = () => {
						if (get(showBackgroudColorLayer)) handleBackgroundColorChange(get(color));
						else handleTextColorChange(get(color));
					};
					template_effect(() => set_style(div_9, `background-color: ${get(color) ?? ""};`));
					append($$anchor$4, div_9);
				});
				reset(div_8);
				append($$anchor$3, div_8);
			};
			if_block(node_6, ($$render) => {
				if (get(showBackgroudColorLayer) || get(showTextColorLayer)) $$render(consequent_1);
			});
			var node_7 = sibling(node_6, 2);
			var consequent_2 = ($$anchor$3) => {
				var div_10 = root_9();
				each(div_10, 21, () => editorTextSizes, index, ($$anchor$4, e$1) => {
					var button_3 = root_10();
					button_3.__click = () => {
						handleFontSizeChange(`${get(e$1).id}px`);
						set(showTextSizeLayer, false);
					};
					var text_2 = child(button_3, true);
					reset(button_3);
					template_effect(() => set_text(text_2, get(e$1).name));
					append($$anchor$4, button_3);
				});
				reset(div_10);
				append($$anchor$3, div_10);
			};
			if_block(node_7, ($$render) => {
				if (get(showTextSizeLayer)) $$render(consequent_2);
			});
			reset(div_2);
			event("blur", input, (ev) => {
				if (avoidCloseOnBlur) {
					avoidCloseOnBlur = false;
					ev.target.focus();
					return;
				}
				set(showTableLayer, false);
				set(showTextColorLayer, false);
				set(showBackgroudColorLayer, false);
				set(showTextSizeLayer, false);
			});
			append($$anchor$2, div_2);
		};
		if_block(node_4, ($$render) => {
			if (get(showLayer)) $$render(consequent_3);
		});
		var node_8 = sibling(node_4, 2);
		var consequent_4 = ($$anchor$2) => {
			var fragment_1 = comment();
			each(first_child(fragment_1), 17, () => tableButtons, index, ($$anchor$3, item) => {
				var button_4 = root_12();
				button_4.__click = function(...$$args) {
					get(item).action?.apply(this, $$args);
				};
				var text_3 = child(button_4, true);
				reset(button_4);
				template_effect(() => {
					button_4.disabled = !get(editor);
					set_attribute(button_4, "aria-label", get(item).label);
					set_text(text_3, get(item).icon);
				});
				append($$anchor$3, button_4);
			});
			append($$anchor$2, fragment_1);
		};
		if_block(node_8, ($$render) => {
			if (get(isInTable)) $$render(consequent_4);
		});
		reset(div_1);
		var div_11 = sibling(div_1, 2);
		bind_this(child(div_11), ($$value) => set(editorRoot, $$value), () => get(editorRoot));
		reset(div_11);
		bind_this(div_11, ($$value) => set(editorContainer, $$value), () => get(editorContainer));
		append($$anchor$1, fragment);
	};
	var alternate = ($$anchor$1) => {
		append($$anchor$1, root_13());
	};
	if_block(node_1, ($$render) => {
		if (browser) $$render(consequent_5);
		else $$render(alternate, false);
	});
	reset(div);
	template_effect(() => set_class(div, 1, `flex flex-col ${$$props.css ?? ""}`, "aME"));
	append($$anchor, div);
	pop();
}
delegate([
	"click",
	"keydown",
	"mousedown"
]);
const defaultTexts = {
	label: {
		h: "hue channel",
		s: "saturation channel",
		v: "brightness channel",
		r: "red channel",
		g: "green channel",
		b: "blue channel",
		a: "alpha channel",
		hex: "hex color",
		withoutColor: "without color"
	},
	color: {
		rgb: "rgb",
		hsv: "hsv",
		hex: "hex"
	},
	changeTo: "change to ",
	swatch: {
		ariaTitle: "saved colors",
		ariaLabel: (color) => `select color: ${color}`
	}
};
const FOCUSABLE_ELEMENTS = "a[href], area[href], input:not([type='hidden']):not([disabled]), select:not([disabled]), textarea:not([disabled]), button:not([disabled]), iframe, object, embed, *[tabindex], *[contenteditable]";
function trapFocusListener(trapFocusElement) {
	return function(event$1) {
		if (event$1.target === window) return;
		const eventTarget = event$1.target;
		if (!trapFocusElement.contains(eventTarget)) return;
		const focusable = trapFocusElement.querySelectorAll(FOCUSABLE_ELEMENTS);
		const first = focusable[0];
		const last = focusable[focusable.length - 1];
		function isNext(event$2) {
			return event$2.code === "Tab" && !event$2.shiftKey;
		}
		function isPrevious(event$2) {
			return event$2.code === "Tab" && event$2.shiftKey;
		}
		if (isNext(event$1) && event$1.target === last) {
			event$1.preventDefault();
			first.focus();
		} else if (isPrevious(event$1) && event$1.target === first) {
			event$1.preventDefault();
			last.focus();
		}
	};
}
const trapFocus = (node) => {
	const first = node.querySelector(FOCUSABLE_ELEMENTS);
	if (first) first.focus();
	const listener = trapFocusListener(node);
	document.addEventListener("keydown", listener);
	return { destroy() {
		document.removeEventListener("keydown", listener);
	} };
};
var r = {
	grad: .9,
	turn: 360,
	rad: 360 / (2 * Math.PI)
}, t = function(r$1) {
	return "string" == typeof r$1 ? r$1.length > 0 : "number" == typeof r$1;
}, n = function(r$1, t$1, n$1) {
	return void 0 === t$1 && (t$1 = 0), void 0 === n$1 && (n$1 = Math.pow(10, t$1)), Math.round(n$1 * r$1) / n$1 + 0;
}, e = function(r$1, t$1, n$1) {
	return void 0 === t$1 && (t$1 = 0), void 0 === n$1 && (n$1 = 1), r$1 > n$1 ? n$1 : r$1 > t$1 ? r$1 : t$1;
}, u = function(r$1) {
	return (r$1 = isFinite(r$1) ? r$1 % 360 : 0) > 0 ? r$1 : r$1 + 360;
}, a = function(r$1) {
	return {
		r: e(r$1.r, 0, 255),
		g: e(r$1.g, 0, 255),
		b: e(r$1.b, 0, 255),
		a: e(r$1.a)
	};
}, o = function(r$1) {
	return {
		r: n(r$1.r),
		g: n(r$1.g),
		b: n(r$1.b),
		a: n(r$1.a, 3)
	};
}, i = /^#([0-9a-f]{3,8})$/i, s = function(r$1) {
	var t$1 = r$1.toString(16);
	return t$1.length < 2 ? "0" + t$1 : t$1;
}, h = function(r$1) {
	var t$1 = r$1.r, n$1 = r$1.g, e$1 = r$1.b, u$1 = r$1.a, a$1 = Math.max(t$1, n$1, e$1), o$1 = a$1 - Math.min(t$1, n$1, e$1), i$1 = o$1 ? a$1 === t$1 ? (n$1 - e$1) / o$1 : a$1 === n$1 ? 2 + (e$1 - t$1) / o$1 : 4 + (t$1 - n$1) / o$1 : 0;
	return {
		h: 60 * (i$1 < 0 ? i$1 + 6 : i$1),
		s: a$1 ? o$1 / a$1 * 100 : 0,
		v: a$1 / 255 * 100,
		a: u$1
	};
}, b = function(r$1) {
	var t$1 = r$1.h, n$1 = r$1.s, e$1 = r$1.v, u$1 = r$1.a;
	t$1 = t$1 / 360 * 6, n$1 /= 100, e$1 /= 100;
	var a$1 = Math.floor(t$1), o$1 = e$1 * (1 - n$1), i$1 = e$1 * (1 - (t$1 - a$1) * n$1), s$1 = e$1 * (1 - (1 - t$1 + a$1) * n$1), h$1 = a$1 % 6;
	return {
		r: 255 * [
			e$1,
			i$1,
			o$1,
			o$1,
			s$1,
			e$1
		][h$1],
		g: 255 * [
			s$1,
			e$1,
			e$1,
			i$1,
			o$1,
			o$1
		][h$1],
		b: 255 * [
			o$1,
			o$1,
			s$1,
			e$1,
			e$1,
			i$1
		][h$1],
		a: u$1
	};
}, g = function(r$1) {
	return {
		h: u(r$1.h),
		s: e(r$1.s, 0, 100),
		l: e(r$1.l, 0, 100),
		a: e(r$1.a)
	};
}, d = function(r$1) {
	return {
		h: n(r$1.h),
		s: n(r$1.s),
		l: n(r$1.l),
		a: n(r$1.a, 3)
	};
}, f = function(r$1) {
	return b((n$1 = (t$1 = r$1).s, {
		h: t$1.h,
		s: (n$1 *= ((e$1 = t$1.l) < 50 ? e$1 : 100 - e$1) / 100) > 0 ? 2 * n$1 / (e$1 + n$1) * 100 : 0,
		v: e$1 + n$1,
		a: t$1.a
	}));
	var t$1, n$1, e$1;
}, c = function(r$1) {
	return {
		h: (t$1 = h(r$1)).h,
		s: (u$1 = (200 - (n$1 = t$1.s)) * (e$1 = t$1.v) / 100) > 0 && u$1 < 200 ? n$1 * e$1 / 100 / (u$1 <= 100 ? u$1 : 200 - u$1) * 100 : 0,
		l: u$1 / 2,
		a: t$1.a
	};
	var t$1, n$1, e$1, u$1;
}, l = /^hsla?\(\s*([+-]?\d*\.?\d+)(deg|rad|grad|turn)?\s*,\s*([+-]?\d*\.?\d+)%\s*,\s*([+-]?\d*\.?\d+)%\s*(?:,\s*([+-]?\d*\.?\d+)(%)?\s*)?\)$/i, p = /^hsla?\(\s*([+-]?\d*\.?\d+)(deg|rad|grad|turn)?\s+([+-]?\d*\.?\d+)%\s+([+-]?\d*\.?\d+)%\s*(?:\/\s*([+-]?\d*\.?\d+)(%)?\s*)?\)$/i, v = /^rgba?\(\s*([+-]?\d*\.?\d+)(%)?\s*,\s*([+-]?\d*\.?\d+)(%)?\s*,\s*([+-]?\d*\.?\d+)(%)?\s*(?:,\s*([+-]?\d*\.?\d+)(%)?\s*)?\)$/i, m = /^rgba?\(\s*([+-]?\d*\.?\d+)(%)?\s+([+-]?\d*\.?\d+)(%)?\s+([+-]?\d*\.?\d+)(%)?\s*(?:\/\s*([+-]?\d*\.?\d+)(%)?\s*)?\)$/i, y = {
	string: [
		[function(r$1) {
			var t$1 = i.exec(r$1);
			return t$1 ? (r$1 = t$1[1]).length <= 4 ? {
				r: parseInt(r$1[0] + r$1[0], 16),
				g: parseInt(r$1[1] + r$1[1], 16),
				b: parseInt(r$1[2] + r$1[2], 16),
				a: 4 === r$1.length ? n(parseInt(r$1[3] + r$1[3], 16) / 255, 2) : 1
			} : 6 === r$1.length || 8 === r$1.length ? {
				r: parseInt(r$1.substr(0, 2), 16),
				g: parseInt(r$1.substr(2, 2), 16),
				b: parseInt(r$1.substr(4, 2), 16),
				a: 8 === r$1.length ? n(parseInt(r$1.substr(6, 2), 16) / 255, 2) : 1
			} : null : null;
		}, "hex"],
		[function(r$1) {
			var t$1 = v.exec(r$1) || m.exec(r$1);
			return t$1 ? t$1[2] !== t$1[4] || t$1[4] !== t$1[6] ? null : a({
				r: Number(t$1[1]) / (t$1[2] ? 100 / 255 : 1),
				g: Number(t$1[3]) / (t$1[4] ? 100 / 255 : 1),
				b: Number(t$1[5]) / (t$1[6] ? 100 / 255 : 1),
				a: void 0 === t$1[7] ? 1 : Number(t$1[7]) / (t$1[8] ? 100 : 1)
			}) : null;
		}, "rgb"],
		[function(t$1) {
			var n$1 = l.exec(t$1) || p.exec(t$1);
			if (!n$1) return null;
			var e$1, u$1;
			return f(g({
				h: (e$1 = n$1[1], u$1 = n$1[2], void 0 === u$1 && (u$1 = "deg"), Number(e$1) * (r[u$1] || 1)),
				s: Number(n$1[3]),
				l: Number(n$1[4]),
				a: void 0 === n$1[5] ? 1 : Number(n$1[5]) / (n$1[6] ? 100 : 1)
			}));
		}, "hsl"]
	],
	object: [
		[function(r$1) {
			var n$1 = r$1.r, e$1 = r$1.g, u$1 = r$1.b, o$1 = r$1.a, i$1 = void 0 === o$1 ? 1 : o$1;
			return t(n$1) && t(e$1) && t(u$1) ? a({
				r: Number(n$1),
				g: Number(e$1),
				b: Number(u$1),
				a: Number(i$1)
			}) : null;
		}, "rgb"],
		[function(r$1) {
			var n$1 = r$1.h, e$1 = r$1.s, u$1 = r$1.l, a$1 = r$1.a, o$1 = void 0 === a$1 ? 1 : a$1;
			if (!t(n$1) || !t(e$1) || !t(u$1)) return null;
			return f(g({
				h: Number(n$1),
				s: Number(e$1),
				l: Number(u$1),
				a: Number(o$1)
			}));
		}, "hsl"],
		[function(r$1) {
			var n$1 = r$1.h, a$1 = r$1.s, o$1 = r$1.v, i$1 = r$1.a, s$1 = void 0 === i$1 ? 1 : i$1;
			if (!t(n$1) || !t(a$1) || !t(o$1)) return null;
			return b(function(r$2) {
				return {
					h: u(r$2.h),
					s: e(r$2.s, 0, 100),
					v: e(r$2.v, 0, 100),
					a: e(r$2.a)
				};
			}({
				h: Number(n$1),
				s: Number(a$1),
				v: Number(o$1),
				a: Number(s$1)
			}));
		}, "hsv"]
	]
}, N = function(r$1, t$1) {
	for (var n$1 = 0; n$1 < t$1.length; n$1++) {
		var e$1 = t$1[n$1][0](r$1);
		if (e$1) return [e$1, t$1[n$1][1]];
	}
	return [null, void 0];
}, x = function(r$1) {
	return "string" == typeof r$1 ? N(r$1.trim(), y.string) : "object" == typeof r$1 && null !== r$1 ? N(r$1, y.object) : [null, void 0];
}, M = function(r$1, t$1) {
	var n$1 = c(r$1);
	return {
		h: n$1.h,
		s: e(n$1.s + 100 * t$1, 0, 100),
		l: n$1.l,
		a: n$1.a
	};
}, H = function(r$1) {
	return (299 * r$1.r + 587 * r$1.g + 114 * r$1.b) / 1e3 / 255;
}, $ = function(r$1, t$1) {
	var n$1 = c(r$1);
	return {
		h: n$1.h,
		s: n$1.s,
		l: e(n$1.l + 100 * t$1, 0, 100),
		a: n$1.a
	};
}, j = function() {
	function r$1(r$2) {
		this.parsed = x(r$2)[0], this.rgba = this.parsed || {
			r: 0,
			g: 0,
			b: 0,
			a: 1
		};
	}
	return r$1.prototype.isValid = function() {
		return null !== this.parsed;
	}, r$1.prototype.brightness = function() {
		return n(H(this.rgba), 2);
	}, r$1.prototype.isDark = function() {
		return H(this.rgba) < .5;
	}, r$1.prototype.isLight = function() {
		return H(this.rgba) >= .5;
	}, r$1.prototype.toHex = function() {
		return r$2 = o(this.rgba), t$1 = r$2.r, e$1 = r$2.g, u$1 = r$2.b, i$1 = (a$1 = r$2.a) < 1 ? s(n(255 * a$1)) : "", "#" + s(t$1) + s(e$1) + s(u$1) + i$1;
		var r$2, t$1, e$1, u$1, a$1, i$1;
	}, r$1.prototype.toRgb = function() {
		return o(this.rgba);
	}, r$1.prototype.toRgbString = function() {
		return r$2 = o(this.rgba), t$1 = r$2.r, n$1 = r$2.g, e$1 = r$2.b, (u$1 = r$2.a) < 1 ? "rgba(" + t$1 + ", " + n$1 + ", " + e$1 + ", " + u$1 + ")" : "rgb(" + t$1 + ", " + n$1 + ", " + e$1 + ")";
		var r$2, t$1, n$1, e$1, u$1;
	}, r$1.prototype.toHsl = function() {
		return d(c(this.rgba));
	}, r$1.prototype.toHslString = function() {
		return r$2 = d(c(this.rgba)), t$1 = r$2.h, n$1 = r$2.s, e$1 = r$2.l, (u$1 = r$2.a) < 1 ? "hsla(" + t$1 + ", " + n$1 + "%, " + e$1 + "%, " + u$1 + ")" : "hsl(" + t$1 + ", " + n$1 + "%, " + e$1 + "%)";
		var r$2, t$1, n$1, e$1, u$1;
	}, r$1.prototype.toHsv = function() {
		return r$2 = h(this.rgba), {
			h: n(r$2.h),
			s: n(r$2.s),
			v: n(r$2.v),
			a: n(r$2.a, 3)
		};
		var r$2;
	}, r$1.prototype.invert = function() {
		return w({
			r: 255 - (r$2 = this.rgba).r,
			g: 255 - r$2.g,
			b: 255 - r$2.b,
			a: r$2.a
		});
		var r$2;
	}, r$1.prototype.saturate = function(r$2) {
		return void 0 === r$2 && (r$2 = .1), w(M(this.rgba, r$2));
	}, r$1.prototype.desaturate = function(r$2) {
		return void 0 === r$2 && (r$2 = .1), w(M(this.rgba, -r$2));
	}, r$1.prototype.grayscale = function() {
		return w(M(this.rgba, -1));
	}, r$1.prototype.lighten = function(r$2) {
		return void 0 === r$2 && (r$2 = .1), w($(this.rgba, r$2));
	}, r$1.prototype.darken = function(r$2) {
		return void 0 === r$2 && (r$2 = .1), w($(this.rgba, -r$2));
	}, r$1.prototype.rotate = function(r$2) {
		return void 0 === r$2 && (r$2 = 15), this.hue(this.hue() + r$2);
	}, r$1.prototype.alpha = function(r$2) {
		return "number" == typeof r$2 ? w({
			r: (t$1 = this.rgba).r,
			g: t$1.g,
			b: t$1.b,
			a: r$2
		}) : n(this.rgba.a, 3);
		var t$1;
	}, r$1.prototype.hue = function(r$2) {
		var t$1 = c(this.rgba);
		return "number" == typeof r$2 ? w({
			h: r$2,
			s: t$1.s,
			l: t$1.l,
			a: t$1.a
		}) : n(t$1.h);
	}, r$1.prototype.isEqual = function(r$2) {
		return this.toHex() === w(r$2).toHex();
	}, r$1;
}(), w = function(r$1) {
	return r$1 instanceof j ? r$1 : new j(r$1);
};
var root_1$5 = from_html(`<input type="hidden"/>`);
var root$10 = from_html(`<div role="slider" tabindex="0"><div class="track aNk"></div> <div class="thumb aNk"></div></div> <!>`, 1);
function Slider($$anchor, $$props) {
	push($$props, true);
	let min = prop($$props, "min", 3, 0), max = prop($$props, "max", 3, 100), step = prop($$props, "step", 3, 1), value = prop($$props, "value", 15, 50), ariaValueText = prop($$props, "ariaValueText", 3, (current) => current.toString()), direction = prop($$props, "direction", 3, "horizontal"), reverse = prop($$props, "reverse", 3, false), keyboardOnly = prop($$props, "keyboardOnly", 3, false), slider = prop($$props, "slider", 7), isDragging = prop($$props, "isDragging", 7, false);
	const _min = user_derived(() => typeof min() === "string" ? parseFloat(min()) : min());
	const _max = user_derived(() => typeof max() === "string" ? parseFloat(max()) : max());
	const _step = user_derived(() => typeof step() === "string" ? parseFloat(step()) : step());
	function bound(value$1) {
		const ratio = 1 / get(_step);
		const rounded = Math.round(value$1 * ratio) / ratio;
		return Math.max(get(_min), Math.min(get(_max), rounded));
	}
	function keyHandler(e$1) {
		const inc = e$1.shiftKey ? get(_step) * 10 : get(_step);
		if (e$1.key === "ArrowUp" || e$1.key === "ArrowRight") {
			value(value() + inc);
			e$1.preventDefault();
		} else if (e$1.key === "ArrowDown" || e$1.key === "ArrowLeft") {
			value(value() - inc);
			e$1.preventDefault();
		} else if (e$1.key === "Home") {
			value(get(_min));
			e$1.preventDefault();
		} else if (e$1.key === "End") {
			value(get(_max));
			e$1.preventDefault();
		} else if (e$1.key === "PageUp") {
			value(value() + get(_step) * 10);
			e$1.preventDefault();
		} else if (e$1.key === "PageDown") {
			value(value() - get(_step) * 10);
			e$1.preventDefault();
		}
		value(bound(value()));
		$$props.onInput?.(value());
	}
	const config = {
		horizontal: {
			clientSize: "clientWidth",
			offset: "left",
			client: "clientX"
		},
		vertical: {
			clientSize: "clientHeight",
			offset: "top",
			client: "clientY"
		}
	};
	function updateValue(e$1) {
		const clientWidth = slider()?.[config[direction()].clientSize] || 120;
		const sliderOffsetX = slider()?.getBoundingClientRect()[config[direction()].offset] || 0;
		let offsetX = e$1[config[direction()].client] - sliderOffsetX;
		if (direction() === "vertical") offsetX = -1 * offsetX + clientWidth;
		if (reverse()) value(get(_max) - offsetX / clientWidth * (get(_max) - get(_min)));
		else value(offsetX / clientWidth * (get(_max) - get(_min)) + get(_min));
		value(bound(value()));
		$$props.onInput?.(value());
	}
	function jump(e$1) {
		updateValue(e$1);
		isDragging(true);
	}
	function drag(e$1) {
		if (isDragging()) updateValue(e$1);
	}
	function endDrag() {
		isDragging(false);
	}
	function touch(e$1) {
		e$1.preventDefault();
		updateValue({
			clientX: e$1.changedTouches[0].clientX,
			clientY: e$1.changedTouches[0].clientY
		});
	}
	const position = user_derived(() => ((value() - get(_min)) / (get(_max) - get(_min)) * 1).toFixed(4));
	var fragment = root$10();
	event("mousemove", $window, drag);
	event("mouseup", $window, endDrag);
	var div = first_child(fragment);
	let classes;
	div.__keydown = keyHandler;
	div.__mousedown = function(...$$args) {
		(keyboardOnly() ? void 0 : jump)?.apply(this, $$args);
	};
	div.__touchstart = function(...$$args) {
		(keyboardOnly() ? void 0 : touch)?.apply(this, $$args);
	};
	div.__touchmove = function(...$$args) {
		(keyboardOnly() ? void 0 : touch)?.apply(this, $$args);
	};
	div.__touchend = function(...$$args) {
		(keyboardOnly() ? void 0 : touch)?.apply(this, $$args);
	};
	let styles;
	bind_this(div, ($$value) => slider($$value), () => slider());
	var node = sibling(div, 2);
	var consequent = ($$anchor$1) => {
		var input = root_1$5();
		remove_input_defaults(input);
		template_effect(() => {
			set_attribute(input, "name", $$props.name);
			set_value(input, value());
		});
		append($$anchor$1, input);
	};
	if_block(node, ($$render) => {
		if ($$props.name) $$render(consequent);
	});
	template_effect(($0) => {
		classes = set_class(div, 1, "slider aNk", null, classes, { reverse: reverse() });
		set_attribute(div, "aria-orientation", direction());
		set_attribute(div, "aria-valuemax", get(_max));
		set_attribute(div, "aria-valuemin", get(_min));
		set_attribute(div, "aria-valuenow", value());
		set_attribute(div, "aria-valuetext", $0);
		set_attribute(div, "aria-label", $$props.ariaLabel);
		set_attribute(div, "aria-labelledby", $$props.ariaLabelledBy);
		set_attribute(div, "aria-controls", $$props.ariaControls);
		styles = set_style(div, "", styles, { "--position": get(position) });
	}, [() => ariaValueText()(value())]);
	append($$anchor, fragment);
	pop();
}
delegate([
	"keydown",
	"mousedown",
	"touchstart",
	"touchmove",
	"touchend"
]);
var root$9 = from_html(`<div class="picker aNd"><!> <div class="s aNd"><!></div> <div class="v aNd"><!></div></div>`);
function Picker($$anchor, $$props) {
	push($$props, true);
	let s$1 = prop($$props, "s", 15), v$1 = prop($$props, "v", 15);
	let picker = state(void 0);
	let isMouseDown = false;
	let pos = state(proxy({
		x: 100,
		y: 0
	}));
	let pickerColorBg = user_derived(() => w({
		h: $$props.h,
		s: 100,
		v: 100,
		a: 1
	}).toHex());
	function clamp(value, min, max) {
		return Math.min(Math.max(min, value), max);
	}
	function onClick(e$1) {
		if (!get(picker)) return;
		const { width, left, height, top } = get(picker).getBoundingClientRect();
		const mouse = {
			x: clamp(e$1.clientX - left, 0, width),
			y: clamp(e$1.clientY - top, 0, height)
		};
		s$1(clamp(mouse.x / width, 0, 1) * 100);
		v$1(clamp((height - mouse.y) / height, 0, 1) * 100);
		updateColor();
	}
	function pickerMousedown(e$1) {
		e$1.preventDefault();
		if (e$1.button === 0) {
			isMouseDown = true;
			onClick(e$1);
		}
	}
	function mouseUp() {
		isMouseDown = false;
	}
	function mouseMove(e$1) {
		if (isMouseDown) onClick(e$1);
	}
	function touch(e$1) {
		e$1.preventDefault();
		onClick(e$1.changedTouches[0]);
	}
	user_effect(() => {
		if (typeof s$1() === "number" && typeof v$1() === "number" && get(picker)) set(pos, {
			x: s$1(),
			y: 100 - v$1()
		}, true);
	});
	function updateColor(color = {}) {
		$$props.onInput({
			s: s$1(),
			v: v$1(),
			...color
		});
	}
	var div = root$9();
	event("mouseup", $window, mouseUp);
	event("mousemove", $window, mouseMove);
	div.__mousedown = pickerMousedown;
	div.__touchstart = touch;
	div.__touchmove = touch;
	div.__touchend = touch;
	let styles;
	var node = child(div);
	component(node, () => $$props.components.pickerIndicator, ($$anchor$1, components_pickerIndicator) => {
		components_pickerIndicator($$anchor$1, {
			get pos() {
				return get(pos);
			},
			get isDark() {
				return $$props.isDark;
			}
		});
	});
	var div_1 = sibling(node, 2);
	let styles_1;
	Slider(child(div_1), {
		get value() {
			return s$1();
		},
		onInput: (s$2) => updateColor({ s: s$2 }),
		keyboardOnly: true,
		ariaValueText: (value) => `${value}%`,
		get ariaLabel() {
			return $$props.texts.label.s;
		}
	});
	reset(div_1);
	var div_2 = sibling(div_1, 2);
	let styles_2;
	Slider(child(div_2), {
		get value() {
			return v$1();
		},
		onInput: (v$2) => updateColor({ v: v$2 }),
		keyboardOnly: true,
		ariaValueText: (value) => `${value}%`,
		direction: "vertical",
		get ariaLabel() {
			return $$props.texts.label.v;
		}
	});
	reset(div_2);
	reset(div);
	bind_this(div, ($$value) => set(picker, $$value), () => get(picker));
	template_effect(() => {
		styles = set_style(div, "", styles, { "--picker-color-bg": get(pickerColorBg) });
		styles_1 = set_style(div_1, "", styles_1, { "--pos-y": get(pos).y });
		styles_2 = set_style(div_2, "", styles_2, { "--pos-x": get(pos).x });
	});
	append($$anchor, div);
	pop();
}
delegate([
	"mousedown",
	"touchstart",
	"touchmove",
	"touchend"
]);
var root$8 = from_html(`<label class="aNf"><div class="container aNf"><input type="color" aria-haspopup="dialog" class="aNf"/> <div class="alpha aNf"></div> <div class="color aNf"></div></div> </label>`);
function Input($$anchor, $$props) {
	push($$props, true);
	let labelElement = prop($$props, "labelElement", 15), name = prop($$props, "name", 3, void 0);
	function preventDefault(e$1) {
		e$1.preventDefault();
	}
	var label_1 = root$8();
	label_1.__click = preventDefault;
	label_1.__mousedown = preventDefault;
	var div = child(label_1);
	var input = child(div);
	remove_input_defaults(input);
	input.__click = preventDefault;
	input.__mousedown = preventDefault;
	var div_1 = sibling(input, 4);
	let styles;
	reset(div);
	var text = sibling(div);
	reset(label_1);
	bind_this(label_1, ($$value) => labelElement($$value), () => labelElement());
	template_effect(() => {
		set_attribute(label_1, "dir", $$props.dir);
		set_attribute(input, "name", name());
		set_value(input, $$props.hex);
		styles = set_style(div_1, "", styles, { background: $$props.hex });
		set_text(text, ` ${$$props.label ?? ""}`);
		label_1.dir = label_1.dir;
	});
	append($$anchor, label_1);
	pop();
}
delegate(["click", "mousedown"]);
var root$7 = from_html(`<label class="nullability-checkbox aNh"><div class="aNh"><input type="checkbox" class="aNh"/> <span class="aNh"></span></div> </label>`);
function NullabilityCheckbox($$anchor, $$props) {
	push($$props, true);
	let isUndefined = prop($$props, "isUndefined", 15);
	var label = root$7();
	var div = child(label);
	var input = child(div);
	remove_input_defaults(input);
	next(2);
	reset(div);
	var text = sibling(div);
	reset(label);
	template_effect(() => set_text(text, ` ${$$props.texts.label.withoutColor ?? ""}`));
	bind_checked(input, isUndefined);
	append($$anchor, label);
	pop();
}
var root$6 = from_html(`<div class="picker-indicator aNe"></div>`);
function PickerIndicator($$anchor, $$props) {
	push($$props, true);
	var div = root$6();
	let styles;
	template_effect(() => styles = set_style(div, "", styles, {
		"--pos-x": $$props.pos.x,
		"--pos-y": $$props.pos.y
	}));
	append($$anchor, div);
	pop();
}
var root_2$2 = from_html(`<button type="button" class="swatch aNj"></button>`);
var root_1$4 = from_html(`<div class="swatches aNj"></div>`);
function Swatches($$anchor, $$props) {
	push($$props, true);
	var fragment = comment();
	var node = first_child(fragment);
	var consequent = ($$anchor$1) => {
		var div = root_1$4();
		each(div, 20, () => $$props.swatches, (color) => color, ($$anchor$2, color) => {
			var button = root_2$2();
			button.__click = () => $$props.selectSwatch(color);
			template_effect(($0) => {
				set_style(button, `background: ${color ?? ""}`);
				set_attribute(button, "aria-label", $0);
			}, [() => $$props.texts.swatch.ariaLabel(color)]);
			append($$anchor$2, button);
		});
		reset(div);
		template_effect(() => set_attribute(div, "aria-label", $$props.texts.swatch.ariaTitle));
		append($$anchor$1, div);
	};
	if_block(node, ($$render) => {
		if ($$props.swatches) $$render(consequent);
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click"]);
var root_1$3 = from_html(`<input class="aNg"/>`);
var root_3$2 = from_html(`<input type="number" min="0" max="255" class="aNg"/> <input type="number" min="0" max="255" class="aNg"/> <input type="number" min="0" max="255" class="aNg"/>`, 1);
var root_4$2 = from_html(`<input type="number" min="0" max="360" class="aNg"/> <input type="number" min="0" max="100" class="aNg"/> <input type="number" min="0" max="100" class="aNg"/>`, 1);
var root_5$1 = from_html(`<input type="number" min="0" max="1" step="0.01" class="aNg"/>`);
var root_6$1 = from_html(`<button type="button" class="aNg"><span class="disappear aNg" aria-hidden="true"> </span> <span class="appear aNg"> </span></button>`);
var root_7$1 = from_html(`<div class="button-like aNg"> </div>`);
var root$5 = from_html(`<div class="text-input aNg"><div class="input-container aNg"><!> <!></div> <!></div>`);
function TextInput($$anchor, $$props) {
	push($$props, true);
	let rgb = prop($$props, "rgb", 15), hsv = prop($$props, "hsv", 15), hex = prop($$props, "hex", 15);
	const HEX_COLOR_REGEX = /^#?([A-F0-9]{6}|[A-F0-9]{8})$/i;
	let mode = state(proxy($$props.textInputModes[0] || "hex"));
	let nextMode = user_derived(() => $$props.textInputModes[($$props.textInputModes.indexOf(get(mode)) + 1) % $$props.textInputModes.length]);
	let h$1 = user_derived(() => Math.round(hsv().h));
	let s$1 = user_derived(() => Math.round(hsv().s));
	let v$1 = user_derived(() => Math.round(hsv().v));
	let a$1 = user_derived(() => hsv().a === void 0 ? 1 : Math.round(hsv().a * 100) / 100);
	function updateHex(e$1) {
		const target = e$1.target;
		if (HEX_COLOR_REGEX.test(target.value)) {
			hex(target.value);
			$$props.onInput({ hex: hex() });
		}
	}
	function updateRgb(property) {
		return function(e$1) {
			let value = parseFloat(e$1.target.value);
			rgb({
				...rgb(),
				[property]: isNaN(value) ? 0 : value
			});
			$$props.onInput({ rgb: rgb() });
		};
	}
	function updateHsv(property) {
		return function(e$1) {
			let value = parseFloat(e$1.target.value);
			hsv({
				...hsv(),
				[property]: isNaN(value) ? 0 : value
			});
			$$props.onInput({ hsv: hsv() });
		};
	}
	var div = root$5();
	var div_1 = child(div);
	var node = child(div_1);
	var consequent = ($$anchor$1) => {
		var input = root_1$3();
		remove_input_defaults(input);
		input.__input = updateHex;
		set_style(input, "", {}, { flex: 3 });
		template_effect(() => {
			set_attribute(input, "aria-label", $$props.texts.label.hex);
			set_value(input, hex());
		});
		append($$anchor$1, input);
	};
	var alternate_1 = ($$anchor$1) => {
		var fragment = comment();
		var node_1 = first_child(fragment);
		var consequent_1 = ($$anchor$2) => {
			var fragment_1 = root_3$2();
			var input_1 = first_child(fragment_1);
			remove_input_defaults(input_1);
			var event_handler = user_derived(() => updateRgb("r"));
			input_1.__input = function(...$$args) {
				get(event_handler)?.apply(this, $$args);
			};
			var input_2 = sibling(input_1, 2);
			remove_input_defaults(input_2);
			var event_handler_1 = user_derived(() => updateRgb("g"));
			input_2.__input = function(...$$args) {
				get(event_handler_1)?.apply(this, $$args);
			};
			var input_3 = sibling(input_2, 2);
			remove_input_defaults(input_3);
			var event_handler_2 = user_derived(() => updateRgb("b"));
			input_3.__input = function(...$$args) {
				get(event_handler_2)?.apply(this, $$args);
			};
			template_effect(() => {
				set_attribute(input_1, "aria-label", $$props.texts.label.r);
				set_value(input_1, rgb().r);
				set_attribute(input_2, "aria-label", $$props.texts.label.g);
				set_value(input_2, rgb().g);
				set_attribute(input_3, "aria-label", $$props.texts.label.b);
				set_value(input_3, rgb().b);
			});
			append($$anchor$2, fragment_1);
		};
		var alternate = ($$anchor$2) => {
			var fragment_2 = root_4$2();
			var input_4 = first_child(fragment_2);
			remove_input_defaults(input_4);
			var event_handler_3 = user_derived(() => updateHsv("h"));
			input_4.__input = function(...$$args) {
				get(event_handler_3)?.apply(this, $$args);
			};
			var input_5 = sibling(input_4, 2);
			remove_input_defaults(input_5);
			var event_handler_4 = user_derived(() => updateHsv("s"));
			input_5.__input = function(...$$args) {
				get(event_handler_4)?.apply(this, $$args);
			};
			var input_6 = sibling(input_5, 2);
			remove_input_defaults(input_6);
			var event_handler_5 = user_derived(() => updateHsv("v"));
			input_6.__input = function(...$$args) {
				get(event_handler_5)?.apply(this, $$args);
			};
			template_effect(() => {
				set_attribute(input_4, "aria-label", $$props.texts.label.h);
				set_value(input_4, get(h$1));
				set_attribute(input_5, "aria-label", $$props.texts.label.s);
				set_value(input_5, get(s$1));
				set_attribute(input_6, "aria-label", $$props.texts.label.v);
				set_value(input_6, get(v$1));
			});
			append($$anchor$2, fragment_2);
		};
		if_block(node_1, ($$render) => {
			if (get(mode) === "rgb") $$render(consequent_1);
			else $$render(alternate, false);
		}, true);
		append($$anchor$1, fragment);
	};
	if_block(node, ($$render) => {
		if (get(mode) === "hex") $$render(consequent);
		else $$render(alternate_1, false);
	});
	var node_2 = sibling(node, 2);
	var consequent_2 = ($$anchor$1) => {
		var input_7 = root_5$1();
		remove_input_defaults(input_7);
		var event_handler_6 = user_derived(() => get(mode) === "hsv" ? updateHsv("a") : updateRgb("a"));
		input_7.__input = function(...$$args) {
			get(event_handler_6)?.apply(this, $$args);
		};
		template_effect(() => {
			set_attribute(input_7, "aria-label", $$props.texts.label.a);
			set_value(input_7, get(a$1));
		});
		append($$anchor$1, input_7);
	};
	if_block(node_2, ($$render) => {
		if ($$props.isAlpha) $$render(consequent_2);
	});
	reset(div_1);
	var node_3 = sibling(div_1, 2);
	var consequent_3 = ($$anchor$1) => {
		var button = root_6$1();
		button.__click = () => set(mode, get(nextMode), true);
		var span = child(button);
		var text = child(span, true);
		reset(span);
		var span_1 = sibling(span, 2);
		var text_1 = child(span_1);
		reset(span_1);
		reset(button);
		template_effect(() => {
			set_text(text, $$props.texts.color[get(mode)]);
			set_text(text_1, `${$$props.texts.changeTo ?? ""} ${$$props.texts.color[get(nextMode)] ?? ""}`);
		});
		append($$anchor$1, button);
	};
	var alternate_2 = ($$anchor$1) => {
		var div_2 = root_7$1();
		var text_2 = child(div_2, true);
		reset(div_2);
		template_effect(() => set_text(text_2, $$props.texts.color[get(mode)]));
		append($$anchor$1, div_2);
	};
	if_block(node_3, ($$render) => {
		if ($$props.textInputModes.length > 1) $$render(consequent_3);
		else $$render(alternate_2, false);
	});
	reset(div);
	append($$anchor, div);
	pop();
}
delegate(["input", "click"]);
var root$4 = from_html(`<div aria-label="color picker"><!></div>`);
function Wrapper($$anchor, $$props) {
	push($$props, true);
	let wrapper = prop($$props, "wrapper", 15);
	var div = root$4();
	let classes;
	snippet(child(div), () => $$props.children);
	reset(div);
	bind_this(div, ($$value) => wrapper($$value), () => wrapper());
	template_effect(() => {
		classes = set_class(div, 1, "wrapper aNi", null, classes, { "is-open": $$props.isOpen });
		set_attribute(div, "role", $$props.isDialog ? "dialog" : void 0);
	});
	append($$anchor, div);
	pop();
}
var root_3$1 = from_html(`<input type="hidden"/>`);
var root_6 = from_html(`<div class="a aNb"><!></div>`);
var root_4$1 = from_html(`<!> <!> <div class="h aNb"><!></div> <!> <!> <!> <!>`, 1);
var root$3 = from_html(`<span><!> <!></span>`);
function ColorPicker($$anchor, $$props) {
	push($$props, true);
	let components = prop($$props, "components", 19, () => ({})), label = prop($$props, "label", 3, "Choose a color"), name = prop($$props, "name", 3, void 0), nullable = prop($$props, "nullable", 3, false), rgb = prop($$props, "rgb", 31, () => proxy(nullable() ? null : {
		r: 255,
		g: 0,
		b: 0,
		a: 1
	})), hsv = prop($$props, "hsv", 31, () => proxy(nullable() ? null : {
		h: 0,
		s: 100,
		v: 100,
		a: 1
	})), hex = prop($$props, "hex", 31, () => proxy(nullable() ? null : "#ff0000")), color = prop($$props, "color", 15, null), isDark = prop($$props, "isDark", 15, false), isAlpha = prop($$props, "isAlpha", 3, true), isDialog = prop($$props, "isDialog", 3, true), isOpen = prop($$props, "isOpen", 31, () => !isDialog()), position = prop($$props, "position", 3, "responsive"), dir = prop($$props, "dir", 3, "ltr"), isTextInput = prop($$props, "isTextInput", 3, true), textInputModes = prop($$props, "textInputModes", 19, () => [
		"hex",
		"rgb",
		"hsv"
	]), sliderDirection = prop($$props, "sliderDirection", 3, "vertical"), disableCloseClickOutside = prop($$props, "disableCloseClickOutside", 3, false), a11yColors = prop($$props, "a11yColors", 19, () => [{ bgHex: "#ffffff" }]), a11yLevel = prop($$props, "a11yLevel", 3, "AA"), texts = prop($$props, "texts", 3, void 0), a11yTexts = prop($$props, "a11yTexts", 3, void 0);
	let _rgb = state(proxy({
		r: 255,
		g: 0,
		b: 0,
		a: 1
	}));
	let _hsv = state(proxy({
		h: 0,
		s: 100,
		v: 100,
		a: 1
	}));
	let _hex = state("#ff0000");
	let isUndefined = state(false);
	let _isUndefined = state(proxy(get(isUndefined)));
	let spanElement = state(void 0);
	let labelElement = state(void 0);
	let wrapper = state(void 0);
	let trap = void 0;
	let innerWidth = state(1080);
	let innerHeight = state(720);
	const wrapperPadding = 12;
	const default_components = {
		pickerIndicator: PickerIndicator,
		textInput: TextInput,
		input: Input,
		nullabilityCheckbox: NullabilityCheckbox,
		wrapper: Wrapper
	};
	function getComponents() {
		return {
			...default_components,
			...components()
		};
	}
	function getTexts() {
		return {
			label: {
				...defaultTexts.label,
				...texts()?.label
			},
			color: {
				...defaultTexts.color,
				...texts()?.color
			},
			changeTo: texts()?.changeTo ?? defaultTexts.changeTo,
			swatch: {
				...texts()?.swatch,
				...defaultTexts.swatch
			}
		};
	}
	function mousedown({ target }) {
		if (isDialog()) {
			if (get(labelElement)?.contains(target) || get(labelElement)?.isSameNode(target)) isOpen(!isOpen());
			else if (isOpen() && !get(wrapper)?.contains(target) && !disableCloseClickOutside()) isOpen(false);
		}
	}
	function keyup({ key, target }) {
		if (!isDialog() || !get(labelElement) || !get(spanElement)) return;
		else if (key === "Enter" && get(labelElement).contains(target)) {
			isOpen(!isOpen());
			setTimeout(() => {
				if (get(wrapper)) trap = trapFocus(get(wrapper));
			});
		} else if (key === "Escape" && isOpen()) {
			isOpen(false);
			if (get(spanElement).contains(target)) {
				get(labelElement)?.focus();
				trap?.destroy();
			}
		}
	}
	function selectSwatch(color$1) {
		hex(color$1);
		hsv(w(color$1).toHsv());
		rgb(w(color$1).toRgb());
		set(isUndefined, false);
		updateColor();
	}
	function hasColorChanged() {
		return !(hsv() && rgb() && hsv().h === get(_hsv).h && hsv().s === get(_hsv).s && hsv().v === get(_hsv).v && hsv().a === get(_hsv).a && rgb().r === get(_rgb).r && rgb().g === get(_rgb).g && rgb().b === get(_rgb).b && rgb().a === get(_rgb).a && hex() === get(_hex));
	}
	function updateColor() {
		if (get(isUndefined) && !get(_isUndefined)) {
			set(_isUndefined, true);
			hsv(null);
			rgb(null);
			hex(null);
			$$props.onInput?.({
				color: color(),
				hsv: hsv(),
				rgb: rgb(),
				hex: hex()
			});
			return;
		} else if (get(_isUndefined) && !get(isUndefined)) {
			set(_isUndefined, false);
			hsv(snapshot(get(_hsv)));
			rgb(snapshot(get(_rgb)));
			hex(snapshot(get(_hex)));
			$$props.onInput?.({
				color: color(),
				hsv: hsv(),
				rgb: rgb(),
				hex: hex()
			});
			return;
		} else if (!hsv() && !rgb() && !hex()) {
			set(isUndefined, set(_isUndefined, true), true);
			$$props.onInput?.({
				color: null,
				hsv: hsv(),
				rgb: rgb(),
				hex: hex()
			});
			return;
		} else if (!hasColorChanged()) return;
		set(isUndefined, false);
		if (hsv() && hsv().a === void 0) hsv({
			...hsv(),
			a: 1
		});
		if (get(_hsv).a === void 0) set(_hsv, {
			...get(_hsv),
			a: 1
		}, true);
		if (rgb() && rgb().a === void 0) rgb({
			...rgb(),
			a: 1
		});
		if (get(_rgb).a === void 0) set(_rgb, {
			...get(_rgb),
			a: 1
		}, true);
		if (hex()?.substring(7) === "ff") hex(hex().substring(0, 7));
		if (get(_hex)?.substring(7) === "ff") set(_hex, get(_hex).substring(0, 7), true);
		if (hsv() && (hsv().h !== get(_hsv).h || hsv().s !== get(_hsv).s || hsv().v !== get(_hsv).v || hsv().a !== get(_hsv).a || !rgb() && !hex())) {
			color(w(hsv()));
			rgb(color().toRgb());
			hex(color().toHex());
		} else if (rgb() && (rgb().r !== get(_rgb).r || rgb().g !== get(_rgb).g || rgb().b !== get(_rgb).b || rgb().a !== get(_rgb).a || !hsv() && !hex())) {
			color(w(rgb()));
			hex(color().toHex());
			hsv(color().toHsv());
		} else if (hex() && (hex() !== get(_hex) || !hsv() && !rgb())) {
			color(w(hex()));
			rgb(color().toRgb());
			hsv(color().toHsv());
		}
		if (color()) isDark(color().isDark());
		if (!hex() || !hsv() || !rgb()) return;
		set(_hsv, snapshot(hsv()), true);
		set(_rgb, snapshot(rgb()), true);
		set(_hex, hex(), true);
		set(_isUndefined, get(isUndefined), true);
		$$props.onInput?.({
			color: color(),
			hsv: hsv(),
			rgb: rgb(),
			hex: hex()
		});
	}
	user_effect(() => {
		if (hsv() || rgb() || hex()) updateColor();
	});
	user_effect(() => {
		get(isUndefined);
		updateColor();
	});
	function updateLetter(letter) {
		return (letterValue) => {
			if (!hsv()) {
				set(isUndefined, false);
				set(_isUndefined, false);
				hsv(snapshot(get(_hsv)));
			}
			hsv({
				...hsv(),
				[letter]: letterValue
			});
		};
	}
	function updateLetters(letters) {
		return (color$1) => {
			if (!hsv()) {
				set(isUndefined, false);
				set(_isUndefined, false);
				hsv(snapshot(get(_hsv)));
			}
			hsv({
				...hsv(),
				...Object.fromEntries(letters.map((letter) => [letter, color$1[letter]]))
			});
		};
	}
	async function wrapperBoundaryCheck() {
		await tick();
		if (position() === "fixed" || !isOpen() || !isDialog() || !get(labelElement) || !get(wrapper)) return;
		const wrapperRect = get(wrapper).getBoundingClientRect();
		const labelRect = get(labelElement).getBoundingClientRect();
		if (position() === "responsive" || position() === "responsive-y") if (labelRect.top + wrapperRect.height + wrapperPadding > get(innerHeight)) get(wrapper).style.top = `-${wrapperRect.height + wrapperPadding}px`;
		else get(wrapper).style.top = `${labelRect.height + wrapperPadding}px`;
		if (position() === "responsive" || position() === "responsive-x") if (dir() === "rtl") {
			const isWrapperToLeft = labelRect.left + labelRect.width - wrapperRect.width < 0;
			console.log(isWrapperToLeft, labelRect.left - wrapperRect.width, labelRect.left, wrapperRect.width);
			if (isWrapperToLeft) get(wrapper).style.left = `0px`;
			else get(wrapper).style.left = `${labelRect.width - wrapperRect.width}px`;
		} else if (labelRect.left + wrapperRect.width > get(innerWidth)) get(wrapper).style.left = `${labelRect.width - wrapperRect.width}px`;
		else get(wrapper).style.left = `0px`;
	}
	user_effect(() => {
		if (get(innerWidth) && get(innerHeight) && isOpen()) wrapperBoundaryCheck();
	});
	const CPComponents = user_derived(getComponents);
	var span = root$3();
	event("mousedown", $window, mousedown);
	event("keyup", $window, keyup);
	event("scroll", $window, wrapperBoundaryCheck);
	var node = child(span);
	var consequent = ($$anchor$1) => {
		var fragment = comment();
		component(first_child(fragment), () => get(CPComponents).input, ($$anchor$2, CPComponents_input) => {
			CPComponents_input($$anchor$2, {
				get hex() {
					return hex();
				},
				get label() {
					return label();
				},
				get name() {
					return name();
				},
				get dir() {
					return dir();
				},
				get labelElement() {
					return get(labelElement);
				},
				set labelElement($$value) {
					set(labelElement, $$value, true);
				}
			});
		});
		append($$anchor$1, fragment);
	};
	var alternate = ($$anchor$1) => {
		var fragment_1 = comment();
		var node_2 = first_child(fragment_1);
		var consequent_1 = ($$anchor$2) => {
			var input = root_3$1();
			remove_input_defaults(input);
			template_effect(() => {
				set_value(input, hex());
				set_attribute(input, "name", name());
			});
			append($$anchor$2, input);
		};
		if_block(node_2, ($$render) => {
			if (name()) $$render(consequent_1);
		}, true);
		append($$anchor$1, fragment_1);
	};
	if_block(node, ($$render) => {
		if (isDialog()) $$render(consequent);
		else $$render(alternate, false);
	});
	component(sibling(node, 2), () => get(CPComponents).wrapper, ($$anchor$1, CPComponents_wrapper) => {
		CPComponents_wrapper($$anchor$1, {
			get isOpen() {
				return isOpen();
			},
			get isDialog() {
				return isDialog();
			},
			get wrapper() {
				return get(wrapper);
			},
			set wrapper($$value) {
				set(wrapper, $$value, true);
			},
			children: ($$anchor$2, $$slotProps) => {
				var fragment_2 = root_4$1();
				var node_4 = first_child(fragment_2);
				var consequent_2 = ($$anchor$3) => {
					var fragment_3 = comment();
					var node_5 = first_child(fragment_3);
					{
						let $0 = user_derived(getTexts);
						component(node_5, () => get(CPComponents).nullabilityCheckbox, ($$anchor$4, CPComponents_nullabilityCheckbox) => {
							CPComponents_nullabilityCheckbox($$anchor$4, {
								get texts() {
									return get($0);
								},
								get isUndefined() {
									return get(isUndefined);
								},
								set isUndefined($$value) {
									set(isUndefined, $$value, true);
								}
							});
						});
					}
					append($$anchor$3, fragment_3);
				};
				if_block(node_4, ($$render) => {
					if (nullable()) $$render(consequent_2);
				});
				var node_6 = sibling(node_4, 2);
				{
					let $0 = user_derived(getComponents);
					let $1 = user_derived(() => hsv()?.h ?? get(_hsv).h);
					let $2 = user_derived(() => hsv()?.s ?? get(_hsv).s);
					let $3 = user_derived(() => hsv()?.v ?? get(_hsv).v);
					let $4 = user_derived(() => updateLetters(["s", "v"]));
					let $5 = user_derived(getTexts);
					Picker(node_6, {
						get components() {
							return get($0);
						},
						get h() {
							return get($1);
						},
						get s() {
							return get($2);
						},
						get v() {
							return get($3);
						},
						get onInput() {
							return get($4);
						},
						get isDark() {
							return isDark();
						},
						get texts() {
							return get($5);
						}
					});
				}
				var div = sibling(node_6, 2);
				var node_7 = child(div);
				{
					let $0 = user_derived(() => hsv()?.h ?? get(_hsv).h);
					let $1 = user_derived(() => updateLetter("h"));
					let $2 = user_derived(() => sliderDirection() === "vertical");
					Slider(node_7, {
						min: 0,
						max: 360,
						step: 1,
						get value() {
							return get($0);
						},
						get onInput() {
							return get($1);
						},
						get direction() {
							return sliderDirection();
						},
						get reverse() {
							return get($2);
						},
						get ariaLabel() {
							return getTexts().label.h;
						}
					});
				}
				reset(div);
				var node_8 = sibling(div, 2);
				var consequent_3 = ($$anchor$3) => {
					var div_1 = root_6();
					let styles;
					var node_9 = child(div_1);
					{
						let $0 = user_derived(() => hsv()?.a ?? get(_hsv).a);
						let $1 = user_derived(() => updateLetter("a"));
						let $2 = user_derived(() => sliderDirection() === "vertical");
						Slider(node_9, {
							min: 0,
							max: 1,
							step: .01,
							get value() {
								return get($0);
							},
							get onInput() {
								return get($1);
							},
							get direction() {
								return sliderDirection();
							},
							get reverse() {
								return get($2);
							},
							get ariaLabel() {
								return getTexts().label.a;
							}
						});
					}
					reset(div_1);
					template_effect(($0) => styles = set_style(div_1, "", styles, $0), [() => ({ "--alphaless-color": (hex() ? hex() : get(_hex)).substring(0, 7) })]);
					append($$anchor$3, div_1);
				};
				if_block(node_8, ($$render) => {
					if (isAlpha()) $$render(consequent_3);
				});
				var node_10 = sibling(node_8, 2);
				var consequent_4 = ($$anchor$3) => {
					{
						let $0 = user_derived(getTexts);
						Swatches($$anchor$3, {
							get swatches() {
								return $$props.swatches;
							},
							selectSwatch,
							get texts() {
								return get($0);
							}
						});
					}
				};
				if_block(node_10, ($$render) => {
					if ($$props.swatches && $$props.swatches.length > 0) $$render(consequent_4);
				});
				var node_11 = sibling(node_10, 2);
				var consequent_5 = ($$anchor$3) => {
					var fragment_5 = comment();
					var node_12 = first_child(fragment_5);
					{
						let $0 = user_derived(() => hex() ?? get(_hex));
						let $1 = user_derived(() => rgb() ?? get(_rgb));
						let $2 = user_derived(() => hsv() ?? get(_hsv));
						let $3 = user_derived(getTexts);
						component(node_12, () => get(CPComponents).textInput, ($$anchor$4, CPComponents_textInput) => {
							CPComponents_textInput($$anchor$4, {
								get hex() {
									return get($0);
								},
								get rgb() {
									return get($1);
								},
								get hsv() {
									return get($2);
								},
								onInput: (color$1) => {
									if (color$1.hsv) hsv(color$1.hsv);
									else if (color$1.rgb) rgb(color$1.rgb);
									else if (color$1.hex) hex(color$1.hex);
								},
								get isAlpha() {
									return isAlpha();
								},
								get textInputModes() {
									return textInputModes();
								},
								get texts() {
									return get($3);
								}
							});
						});
					}
					append($$anchor$3, fragment_5);
				};
				if_block(node_11, ($$render) => {
					if (isTextInput()) $$render(consequent_5);
				});
				var node_13 = sibling(node_11, 2);
				var consequent_6 = ($$anchor$3) => {
					var fragment_6 = comment();
					var node_14 = first_child(fragment_6);
					{
						let $0 = user_derived(getComponents);
						let $1 = user_derived(() => hex() || "#00000000");
						component(node_14, () => get(CPComponents).a11yNotice, ($$anchor$4, CPComponents_a11yNotice) => {
							CPComponents_a11yNotice($$anchor$4, {
								get components() {
									return get($0);
								},
								get a11yColors() {
									return a11yColors();
								},
								get hex() {
									return get($1);
								},
								get a11yTexts() {
									return a11yTexts();
								},
								get a11yLevel() {
									return a11yLevel();
								}
							});
						});
					}
					append($$anchor$3, fragment_6);
				};
				if_block(node_13, ($$render) => {
					if (getComponents().a11yNotice) $$render(consequent_6);
				});
				append($$anchor$2, fragment_2);
			},
			$$slots: { default: true }
		});
	});
	reset(span);
	bind_this(span, ($$value) => set(spanElement, $$value), () => get(spanElement));
	template_effect(() => set_class(span, 1, `color-picker ${sliderDirection() ?? ""}`, "aNb"));
	bind_window_size("innerWidth", ($$value) => set(innerWidth, $$value, true));
	bind_window_size("innerHeight", ($$value) => set(innerHeight, $$value, true));
	append($$anchor, span);
	pop();
}
var dist_default = ColorPicker;
var root_1$2 = from_html(`<div><div></div></div> <div> </div> <div><div></div></div>`, 1);
var root$2 = from_html(`<div><!> <div><div></div></div> <div><div><div></div></div> <div class="w-full flex items-center justify-center"><!></div></div></div>`);
function ColorPicker_1($$anchor, $$props) {
	push($$props, true);
	let saveOn = prop($$props, "saveOn", 15);
	prop($$props, "contentClass", 3, "");
	let cN = user_derived(() => `${components_module_default.input} p-rel` + ($$props.css ? " " + $$props.css : ""));
	let currentColor = state("#FFFFFF");
	let hasInit = false;
	const setColor$1 = (hexColor) => {
		if (saveOn() && $$props.save) saveOn(saveOn()[$$props.save] = hexColor, true);
		if (hexColor) set(currentColor, hexColor, true);
	};
	user_effect(() => {
		if (saveOn() || $$props.save) {
			console.log("change save on:", snapshot(saveOn()));
			untrack(() => {
				set(currentColor, saveOn()[$$props.save] || "#FFFFFF", true);
			});
			hasInit = true;
		}
	});
	var div = root$2();
	var node = child(div);
	var consequent = ($$anchor$1) => {
		var fragment = root_1$2();
		var div_1 = first_child(fragment);
		var div_2 = sibling(div_1, 2);
		var text = child(div_2, true);
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		template_effect(() => {
			set_class(div_1, 1, clsx(components_module_default.input_lab_cell_left), "aN5");
			set_class(div_2, 1, clsx(components_module_default.input_lab), "aN5");
			set_text(text, $$props.label);
			set_class(div_3, 1, clsx(components_module_default.input_lab_cell_right), "aN5");
		});
		append($$anchor$1, fragment);
	};
	if_block(node, ($$render) => {
		if ($$props.label) $$render(consequent);
	});
	var div_4 = sibling(node, 2);
	var div_5 = sibling(div_4, 2);
	var div_6 = child(div_5);
	var div_7 = sibling(div_6, 2);
	dist_default(child(div_7), {
		isAlpha: false,
		textInputModes: [],
		position: "responsive",
		get hex() {
			return get(currentColor);
		},
		onInput: (color) => {
			if (!hasInit) return;
			console.log("Setting color:", color.hex);
			setColor$1(color.hex);
		}
	});
	reset(div_7);
	reset(div_5);
	reset(div);
	template_effect(() => {
		set_class(div, 1, clsx(get(cN)), "aN5");
		set_class(div_4, 1, clsx(components_module_default.input_shadow_layer), "aN5");
		set_class(div_5, 1, `_1 ${components_module_default.input_div} flex items-center justify-center w-full`, "aN5");
		set_class(div_6, 1, clsx(components_module_default.input_div_1), "aN5");
	});
	append($$anchor, div);
	pop();
}
var root_2$1 = from_html(`<div class="flex justify-center w-full"><div class="_1 h-24 w-36 aMD"></div></div>`);
var root_3 = from_html(`<div class="grid grid-cols-24 gap-10 p-4"><!> <!> <!> <!> <div class="col-span-12"><!></div> <div class="mt-12 col-span-24 fs15"><i class="icon-attention"></i> La información se guardará cuando se guarde el producto.</div></div>`);
var root$1 = from_html(`<div class="flex justify-between mt-4"><div></div> <button class="bx-green s1"><i class="icon-plus"></i></button></div> <!> <!>`, 1);
function Atributos($$anchor, $$props) {
	push($$props, true);
	const produtcoAtributosMap = new Map(productoAtributos.map((e$1) => [e$1.id, e$1]));
	let producto = prop($$props, "producto", 15);
	let presentacionForm = state(proxy({}));
	let tempCounter = -1;
	const columns = [
		{
			header: "Atributo",
			getValue: (e$1) => produtcoAtributosMap.get(e$1.at)?.name || ""
		},
		{
			header: "Nombre",
			getValue: (e$1) => e$1.nm
		},
		{
			header: "Precio",
			getValue: (e$1) => e$1.pc ? formatN(e$1.pc / 100, 2) : ""
		},
		{
			header: "Diff. Precio",
			getValue: (e$1) => e$1.pd ? formatN(e$1.pd / 100, 2) : ""
		},
		{
			header: "Color",
			id: "color",
			getValue: (e$1) => e$1.cl
		},
		{
			header: "...",
			cellCss: "px-6 py-1",
			headerCss: "w-42",
			buttonEditHandler(e$1) {
				set(presentacionForm, { ...e$1 }, true);
				openModal(3);
			}
		}
	];
	const presentacionesFiltered = user_derived(() => (producto().Presentaciones || []).filter((x$1) => x$1.ss));
	var fragment = root$1();
	var div = first_child(fragment);
	var button = sibling(child(div), 2);
	button.__click = () => {
		set(presentacionForm, {
			at: producto().AtributosIDs?.[0],
			id: tempCounter,
			ss: 1
		}, true);
		tempCounter--;
		console.log("presentacionForm", get(presentacionForm));
		openModal(3);
	};
	reset(div);
	var node = sibling(div, 2);
	{
		const cellRenderer = ($$anchor$1, record = noop, col = noop) => {
			var fragment_1 = comment();
			var node_1 = first_child(fragment_1);
			var consequent = ($$anchor$2) => {
				var div_1 = root_2$1();
				var div_2 = child(div_1);
				reset(div_1);
				template_effect(() => set_style(div_2, `background-color:${record().cl ?? ""}`));
				append($$anchor$2, div_1);
			};
			if_block(node_1, ($$render) => {
				if (col().id === "color" && record().cl) $$render(consequent);
			});
			append($$anchor$1, fragment_1);
		};
		VTable(node, {
			get columns() {
				return columns;
			},
			css: "mt-6",
			get data() {
				return get(presentacionesFiltered);
			},
			onRowClick: (e$1) => {
				set(presentacionForm, { ...e$1 }, true);
			},
			cellRenderer,
			$$slots: { cellRenderer: true }
		});
	}
	Modal(sibling(node, 2), {
		title: "Producto Presentación",
		id: 3,
		size: 4,
		saveButtonLabel: "Agregar",
		saveIcon: "icon-ok",
		onSave: () => {
			producto(producto().Presentaciones = producto().Presentaciones || [], true);
			const current = producto().Presentaciones.find((x$1) => x$1.id === get(presentacionForm).id);
			if (current) Object.assign(current, get(presentacionForm));
			else producto().Presentaciones.push(get(presentacionForm));
			producto(producto().Presentaciones = [...producto().Presentaciones], true);
			closeAllModals();
		},
		onDelete: () => {
			const current = producto().Presentaciones.find((x$1) => x$1.id === get(presentacionForm).id);
			if (current && get(presentacionForm).id > 0) current.ss = 0;
			else producto(producto().Presentaciones = producto().Presentaciones.filter((x$1) => x$1.id !== get(presentacionForm).id), true);
			producto(producto().Presentaciones = [...producto().Presentaciones], true);
			closeAllModals();
		},
		children: ($$anchor$1, $$slotProps) => {
			var div_3 = root_3();
			var node_3 = child(div_3);
			SearchSelect(node_3, {
				label: "Atributo",
				get saveOn() {
					return get(presentacionForm);
				},
				css: "col-span-12",
				save: "at",
				keyId: "id",
				keyName: "name",
				get options() {
					return productoAtributos;
				}
			});
			var node_4 = sibling(node_3, 2);
			Input$1(node_4, {
				label: "Nombre",
				get saveOn() {
					return get(presentacionForm);
				},
				css: "col-span-12",
				save: "nm"
			});
			var node_5 = sibling(node_4, 2);
			Input$1(node_5, {
				label: "Precio",
				get saveOn() {
					return get(presentacionForm);
				},
				css: "col-span-12",
				save: "pc",
				type: "number",
				baseDecimals: 2
			});
			var node_6 = sibling(node_5, 2);
			Input$1(node_6, {
				label: "Diferencia Precio",
				get saveOn() {
					return get(presentacionForm);
				},
				css: "col-span-12",
				save: "pd",
				type: "number",
				baseDecimals: 2
			});
			var div_4 = sibling(node_6, 2);
			ColorPicker_1(child(div_4), {
				label: "Color",
				get saveOn() {
					return get(presentacionForm);
				},
				save: "cl"
			});
			reset(div_4);
			next(2);
			reset(div_3);
			append($$anchor$1, div_3);
		},
		$$slots: { default: true }
	});
	append($$anchor, fragment);
	pop();
}
delegate(["click"]);
var root_1$1 = from_html(`<div class="_1 px-12 py-10 w-250 min-h-140 aMC" role="button" tabindex="0"><div class="min-h-70 _2 mb-2 aMC"></div> <div class="fs17 ff-semibold"> </div> <div class="fs15"> </div></div>`);
var root_2 = from_html(`<div class="grid grid-cols-12 gap-10 p-6"><!> <!> <!></div>`);
var root = from_html(`<div class="w-full flex flex-wrap gap-12"></div> <!>`, 1);
function CategoriasMarcas($$anchor, $$props) {
	push($$props, true);
	const listas = prop($$props, "listas", 7);
	const categorias = user_derived(() => {
		console.log("listas getted:", snapshot(listas()));
		return listas().ListaRecordsMap.get($$props.origin) || [];
	});
	let form = state(proxy({}));
	const imagesIDs = [
		151,
		152,
		153
	];
	let images = state(proxy([
		{ _id: 151 },
		{ _id: 152 },
		{ _id: 153 }
	]));
	const onSave = async (isDelete) => {
		console.log("form a enviar 1::", snapshot(get(form)), isDelete);
		if ((get(form).Nombre || "").length < 4 || !get(form).ListaID) {
			Notify.failure("Debe colocar un nombre de al menos 4 caracteres.");
			return;
		}
		Loading.standard("Guardando categoría...");
		get(form).Images = get(form).Images || [];
		get(form).ss = isDelete ? 0 : 1;
		for (let i$1 = 0; i$1 < imagesIDs.length; i$1++) {
			const imageID = imagesIDs[i$1];
			if (imagesToUpload.has(imageID)) {
				Loading.change(`Guardando Imagen ${i$1 + 1}`);
				let imageName = (await imagesToUpload.get(imageID)?.())?.imageName || "";
				if (imageName.includes("/")) imageName = imageName.split("/")[1];
				get(form).Images[i$1] = imageName;
			} else get(form).Images[i$1] = get(form).Images?.[i$1] || "";
		}
		console.log("form a enviar 2::", snapshot(get(form)));
		Loading.change("Guardando categoría...");
		let result;
		try {
			result = await postListaRegistros([get(form)]);
		} catch (error) {
			Notify.failure(error);
			return;
		}
		const selected = get(categorias).find((x$1) => x$1.ID === get(form).ID);
		let newCategorias = [...get(categorias)];
		if (selected && isDelete) newCategorias = newCategorias.filter((x$1) => x$1.ID !== get(form).ID);
		else if (selected) Object.assign(selected, get(form));
		else {
			get(form).ID = result[0].NewID;
			newCategorias.push(get(form));
			listas().RecordsMap.set(get(form).ID, get(form));
		}
		console.log("newCategorias", newCategorias.length, "|", get(categorias).length);
		listas().ListaRecordsMap.set($$props.origin, newCategorias);
		listas().ListaRecordsMap = new Map(listas().ListaRecordsMap);
		closeAllModals();
		Loading.remove();
	};
	const newRecord = () => {
		set(form, { ListaID: $$props.origin }, true);
		openModal(2);
	};
	user_effect(() => {
		console.log("form a enviar::", snapshot(get(form)));
	});
	const selectCategoria = (e$1) => {
		set(form, { ...e$1 }, true);
		set(images, imagesIDs.map((_id, i$1) => {
			return {
				_id,
				src: (get(form).Images || [])[i$1] || ""
			};
		}), true);
		console.log("categoría getted:", snapshot(get(form)), snapshot(get(images)));
		openModal(2);
	};
	var $$exports = { newRecord };
	var fragment = root();
	var div = first_child(fragment);
	each(div, 21, () => get(categorias), index, ($$anchor$1, e$1) => {
		var div_1 = root_1$1();
		div_1.__keydown = (ev) => {
			ev.stopPropagation();
			if (ev.key === "Enter" || ev.key === " ") {
				ev.preventDefault();
				selectCategoria(get(e$1));
			}
		};
		div_1.__click = (ev) => {
			ev.stopPropagation();
			selectCategoria(get(e$1));
		};
		var div_2 = sibling(child(div_1), 2);
		var text = child(div_2, true);
		reset(div_2);
		var div_3 = sibling(div_2, 2);
		var text_1 = child(div_3, true);
		reset(div_3);
		reset(div_1);
		template_effect(() => {
			set_text(text, get(e$1).Nombre);
			set_text(text_1, get(e$1).Descripcion);
		});
		append($$anchor$1, div_1);
	});
	reset(div);
	var node = sibling(div, 2);
	{
		let $0 = user_derived(() => get(form).ID > 0 ? () => {
			onSave(true);
		} : void 0);
		Modal(node, {
			title: "CATEGORÍA",
			id: 2,
			size: 6,
			onClose: () => {
				closeModal(2);
			},
			onSave: () => {
				onSave();
			},
			get onDelete() {
				return get($0);
			},
			children: ($$anchor$1, $$slotProps) => {
				var div_4 = root_2();
				var node_1 = child(div_4);
				Input$1(node_1, {
					label: "Nombre",
					css: "col-span-12",
					get saveOn() {
						return get(form);
					},
					save: "Nombre",
					required: true
				});
				var node_2 = sibling(node_1, 2);
				Input$1(node_2, {
					label: "Descripción",
					css: "col-span-12 mb-16",
					get saveOn() {
						return get(form);
					},
					save: "Descripcion"
				});
				each(sibling(node_2, 2), 17, () => get(images), index, ($$anchor$2, image, index$1) => {
					ImageUploader($$anchor$2, {
						clearOnUpload: true,
						get id() {
							return get(image)._id;
						},
						folder: "img-public",
						get src() {
							return get(image).src;
						},
						size: 2,
						cardCss: "w-full h-180 p-4 col-span-4",
						types: ["avif", "webp"],
						saveAPI: "producto-categoria-image",
						hideUploadButton: true,
						hideForm: true,
						onChange: (e$1) => {
							Object.assign(get(image), e$1);
						},
						setDataToSend: (e$1) => {
							e$1.Order = index$1 + 1;
						}
					});
				});
				reset(div_4);
				append($$anchor$1, div_4);
			},
			$$slots: { default: true }
		});
	}
	append($$anchor, fragment);
	return pop($$exports);
}
delegate(["keydown", "click"]);
var root_5 = from_html(`<div class="grid grid-cols-24 items-start gap-x-10 gap-y-10 mt-6 md:mt-16"><!> <div class="col-span-24 md:col-span-9 md:row-span-4"><!></div> <!> <!> <!> <!> <!> <!> <!> <!> <div class="col-span-24 md:col-span-14"><!></div> <!> <!> <div class="ff-bold h3 col-span-24 ml-8">Sub-Unidades</div> <!> <!> <!> <!> <!></div>`);
var root_7 = from_html(`<div class="mt-16 w-full"><!></div>`);
var root_8 = from_html(`<div class="grid grid-cols-2 md:grid-cols-4 items-start gap-x-10 gap-y-10 mt-16"><!> <!></div>`);
var root_4 = from_html(`<!> <!> <!> <!>`, 1);
var root_1 = from_html(`<div class="grid grid-cols-12 md:flex md:flex-row items-center mb-8"><!> <div class="i-search w-full md:w-200 md:ml-12 col-span-5"><div><i class="icon-search"></i></div> <input type="text"/></div> <button class="bx-green ml-auto col-span-7"><i class="icon-plus"></i><span class="hidden md:block">Nuevo</span></button></div> <!> <!> <!> <!>`, 1);
function _page($$anchor, $$props) {
	push($$props, true);
	let filterText = state("");
	const productos = new ProductosService();
	const listas = new ListasCompartidasService([1, 2]);
	let view = state(1);
	let layerView = state(1);
	let productoForm = state(proxy({}));
	let CategoriasLayer = null;
	let MarcasLayer = null;
	let productoColumns = [
		{
			header: "ID",
			css: "c-blue text-center",
			headerCss: "w-48",
			getValue: (e$1) => e$1.ID,
			mobile: {
				order: 1,
				css: "col-span-6",
				icon: "tag",
				render: (e$1) => `<strong>${e$1.ID}</strong>`
			}
		},
		{
			header: "Producto",
			highlight: true,
			getValue: (e$1) => e$1.Nombre,
			mobile: {
				order: 2,
				css: "col-span-18",
				render: (e$1) => `<strong>${e$1.Nombre}</strong>`
			}
		},
		{
			header: "Categorías",
			highlight: true,
			getValue: (e$1) => {
				const nombres = [];
				for (const id of e$1.CategoriasIDs || []) {
					const nombre = listas.get(id)?.Nombre || `Categoría-${id}`;
					nombres.push(nombre);
				}
				return nombres.join(", ");
			},
			mobile: {
				order: 3,
				css: "col-span-24",
				render: (e$1) => {
					const nombres = [];
					for (const id of e$1.CategoriasIDs || []) {
						const nombre = listas.get(id)?.Nombre || `Categoría-${id}`;
						nombres.push(nombre);
					}
					return `<div style="font-size: 0.85rem; color: #666;">${nombres.join(", ") || "Sin categorías"}</div>`;
				}
			}
		},
		{
			header: "Precio",
			css: "text-right",
			getValue: (e$1) => formatN(e$1.Precio / 100, 2),
			mobile: {
				order: 4,
				css: "col-span-8",
				labelLeft: "Precio:",
				render: (e$1) => formatN(e$1.Precio / 100, 2)
			}
		},
		{
			header: "Descuento",
			css: "text-right",
			getValue: (e$1) => e$1.Descuento ? String(e$1.Descuento) + "%" : "",
			mobile: {
				order: 5,
				css: "col-span-8",
				labelLeft: "Desc:",
				render: (e$1) => e$1.Descuento ? `${e$1.Descuento}%` : "-"
			}
		},
		{
			header: "Precio Final",
			css: "text-right",
			getValue: (e$1) => formatN(e$1.PrecioFinal / 100, 2),
			mobile: {
				order: 6,
				css: "col-span-8",
				labelLeft: "Final:",
				icon: "ok",
				render: (e$1) => `<strong>${formatN(e$1.PrecioFinal / 100, 2)}</strong>`
			}
		},
		{
			header: "Sub Unidades",
			css: "text-right",
			getValue: (e$1) => {
				if (!e$1.SbnUnidad) return "";
				return `${e$1.SbnCantidad} x ${e$1.SbnUnidad}`;
			},
			mobile: {
				order: 7,
				css: "col-span-24",
				labelTop: "Sub-unidades",
				render: (e$1) => {
					if (!e$1.SbnUnidad) return "<span style='color: #999;'>-</span>";
					return `<div style="font-size: 0.9rem;">${e$1.SbnCantidad} x ${e$1.SbnUnidad}</div>`;
				}
			}
		}
	];
	const categorias = user_derived(() => {
		return listas.ListaRecordsMap.get(1) || [];
	});
	const onSave = async (isDelete) => {
		if ((get(productoForm).Nombre?.length || 0) < 4) {
			Notify.failure("El nombre debe tener al menos 4 caracteres.");
			return;
		}
		console.log("productor a enviar:", snapshot(get(productoForm)));
		if (isDelete) get(productoForm).ss = 0;
		Loading.standard("Guardando producto...");
		try {
			var productosUpdated = await postProducto([get(productoForm)]);
		} catch (error) {
			Notify.failure(error);
			Loading.remove();
			return;
		}
		Loading.remove();
		const producto = productos.productos.find((x$1) => x$1.ID === get(productoForm).ID);
		if (producto && isDelete) productos.productos = productos.productos.filter((x$1) => x$1.ID !== get(productoForm).ID);
		else if (producto) Object.assign(producto, get(productoForm));
		else {
			get(productoForm).ID = productosUpdated[0].ID;
			productos.productos.unshift(get(productoForm));
			productos.productos = [...productos.productos];
		}
		Core.openSideLayer(0);
	};
	user_effect(() => {
		console.log("productor form:", snapshot(get(productoForm)));
	});
	const deleteProductoImage = async (ImageToDelete) => {
		Loading.standard("Eliminando Imagen...");
		try {
			await POST({
				data: {
					ProductoID: get(productoForm).ID,
					ImageToDelete
				},
				route: "producto-image",
				refreshRoutes: ["productos"]
			});
		} catch (error) {
			Notify.failure(`Error al eliminar imagen: ${error}`);
			Loading.remove();
			return;
		}
		Loading.remove();
		get(productoForm).Images = (get(productoForm).Images || []).filter((e$1) => e$1.n !== ImageToDelete);
	};
	Page($$anchor, {
		sideLayerSize: 780,
		title: "Productos",
		children: ($$anchor$1, $$slotProps) => {
			var fragment_1 = root_1();
			var div = first_child(fragment_1);
			var node = child(div);
			OptionsStrip(node, {
				get selected() {
					return get(view);
				},
				css: "col-span-12 mb-6 md:mb-0",
				options: [
					[1, "Productos"],
					[2, "Categorías"],
					[3, "Marcas"]
				],
				useMobileGrid: true,
				onSelect: (e$1) => {
					Core.openSideLayer(0);
					set(productoForm, { ID: 0 }, true);
					set(view, e$1[0], true);
				}
			});
			var div_1 = sibling(node, 2);
			var input = sibling(child(div_1), 2);
			input.__keyup = (ev) => {
				const value = String(ev.target.value || "");
				throttle(() => {
					set(filterText, value, true);
				}, 150);
			};
			reset(div_1);
			var button = sibling(div_1, 2);
			button.__click = (ev) => {
				ev.stopPropagation();
				if (get(view) === 2) CategoriasLayer?.newRecord();
				else if (get(view) === 3) MarcasLayer?.newRecord();
				else Core.openSideLayer(1);
			};
			reset(div);
			var node_1 = sibling(div, 2);
			var consequent = ($$anchor$2) => {
				Layer($$anchor$2, {
					type: "content",
					children: ($$anchor$3, $$slotProps$1) => {
						{
							let $0 = user_derived(() => get(productoForm)?.ID);
							VTable($$anchor$3, {
								get columns() {
									return productoColumns;
								},
								get data() {
									return productos.productos;
								},
								get filterText() {
									return get(filterText);
								},
								get selected() {
									return get($0);
								},
								isSelected: (e$1, id) => e$1.ID === id,
								getFilterContent: (e$1) => {
									return e$1.Nombre;
								},
								onRowClick: (e$1) => {
									set(productoForm, { ...e$1 }, true);
									get(productoForm).CategoriasIDs = [...e$1.CategoriasIDs || []];
									get(productoForm).Propiedades = [...e$1.Propiedades || []];
									Core.openSideLayer(1);
								},
								mobileCardCss: "mb-2"
							});
						}
					},
					$$slots: { default: true }
				});
			};
			if_block(node_1, ($$render) => {
				if (get(view) === 1) $$render(consequent);
			});
			var node_2 = sibling(node_1, 2);
			{
				let $0 = user_derived(() => get(productoForm)?.Nombre || "");
				Layer(node_2, {
					type: "side",
					css: "px-8 py-8 md:px-16 md:py-10",
					get title() {
						return get($0);
					},
					titleCss: "h2 mb-6",
					contentCss: "px-0 md:px-0",
					id: 1,
					onClose: () => {
						set(productoForm, {}, true);
					},
					onSave: () => {
						onSave();
					},
					onDelete: () => {
						ConfirmWarn("Eliminar Producto", `¿Está seguro que desea eliminar "${get(productoForm).Nombre}"?`, "SI", "NO", () => {
							onSave(true);
						});
					},
					options: [
						[
							1,
							"Información",
							["Informa-", "ción"]
						],
						[2, "Ficha"],
						[
							3,
							"Presentaciones",
							["Presenta-", "ciones"]
						],
						[4, "Fotos"]
					],
					get selected() {
						return get(layerView);
					},
					set selected($$value) {
						set(layerView, $$value, true);
					},
					children: ($$anchor$2, $$slotProps$1) => {
						var fragment_4 = root_4();
						var node_3 = first_child(fragment_4);
						var consequent_1 = ($$anchor$3) => {
							var div_2 = root_5();
							var node_4 = child(div_2);
							Input$1(node_4, {
								label: "Nombre",
								get saveOn() {
									return get(productoForm);
								},
								css: "col-span-24",
								required: true,
								save: "Nombre"
							});
							var div_3 = sibling(node_4, 2);
							var node_5 = child(div_3);
							{
								let $0$1 = user_derived(() => get(productoForm).Image?.n);
								ImageUploader(node_5, {
									saveAPI: "producto-image",
									useConvertAvif: true,
									clearOnUpload: true,
									types: ["avif", "webp"],
									folder: "img-productos",
									size: 2,
									get src() {
										return get($0$1);
									},
									cardCss: "w-full h-180 p-4",
									get imageSource() {
										return get(productoForm)._imageSource;
									},
									setDataToSend: (e$1) => {
										e$1.ProductoID = get(productoForm).ID;
									},
									onChange: (e$1, uploadHandler) => {
										get(productoForm)._imageSource = e$1;
									},
									onUploaded: (imagePath, description) => {
										if (imagePath.includes("/")) imagePath = imagePath.split("/")[1];
										get(productoForm).Image = {
											n: imagePath,
											d: description
										};
										get(productoForm).Images = get(productoForm).Images || [];
										get(productoForm).Images.unshift(get(productoForm).Image);
									}
								});
							}
							reset(div_3);
							var node_6 = sibling(div_3, 2);
							Input$1(node_6, {
								label: "Precio Base",
								get saveOn() {
									return get(productoForm);
								},
								css: "col-span-12 md:col-span-5",
								save: "Precio",
								type: "number",
								baseDecimals: 2,
								get dependencyValue() {
									return get(productoForm).Precio;
								},
								onChange: () => {
									console.log("productoForm", snapshot(get(productoForm)));
									get(productoForm).PrecioFinal = Math.round(get(productoForm).Precio * (100 - (get(productoForm).Descuento || 0)) / 100);
								}
							});
							var node_7 = sibling(node_6, 2);
							Input$1(node_7, {
								label: "Descuento",
								get saveOn() {
									return get(productoForm);
								},
								css: "col-span-12 md:col-span-5",
								save: "Descuento",
								postValue: "%",
								type: "number",
								validator: (v$1) => {
									return !v$1 || v$1 < 100;
								}
							});
							var node_8 = sibling(node_7, 2);
							Input$1(node_8, {
								label: "Precio Final",
								get saveOn() {
									return get(productoForm);
								},
								css: "col-span-12 md:col-span-5",
								get dependencyValue() {
									return get(productoForm).PrecioFinal;
								},
								save: "PrecioFinal",
								type: "number",
								baseDecimals: 2,
								onChange: () => {
									console.log("productoForm", snapshot(get(productoForm)));
									get(productoForm).Precio = Math.round(get(productoForm).PrecioFinal / (100 - (get(productoForm).Descuento || 0)) * 100);
								}
							});
							var node_9 = sibling(node_8, 2);
							SearchSelect(node_9, {
								label: "Moneda",
								get saveOn() {
									return get(productoForm);
								},
								css: "col-span-12 md:col-span-5",
								save: "MonedaID",
								keyId: "i",
								keyName: "v",
								options: [{
									i: 1,
									v: "PEN (S/.)"
								}, {
									i: 2,
									v: "USD ($)"
								}]
							});
							var node_10 = sibling(node_9, 2);
							Input$1(node_10, {
								label: "Peso",
								get saveOn() {
									return get(productoForm);
								},
								css: "col-span-12 md:col-span-5",
								save: "Peso",
								type: "number"
							});
							var node_11 = sibling(node_10, 2);
							SearchSelect(node_11, {
								label: "Unidad",
								get saveOn() {
									return get(productoForm);
								},
								css: "col-span-12 md:col-span-5",
								save: "UnidadID",
								keyId: "i",
								keyName: "v",
								options: [
									{
										i: 1,
										v: "Kg"
									},
									{
										i: 2,
										v: "g"
									},
									{
										i: 3,
										v: "Libras"
									}
								]
							});
							var node_12 = sibling(node_11, 2);
							Input$1(node_12, {
								label: "Volumen",
								get saveOn() {
									return get(productoForm);
								},
								css: "col-span-12 md:col-span-5",
								save: "Volumen",
								type: "number"
							});
							var node_13 = sibling(node_12, 2);
							{
								let $0$1 = user_derived(() => listas.ListaRecordsMap.get(2) || []);
								SearchSelect(node_13, {
									label: "Marca",
									get saveOn() {
										return get(productoForm);
									},
									css: "col-span-12 md:col-span-10 mb-2",
									save: "MarcaID",
									keyId: "ID",
									keyName: "Nombre",
									get options() {
										return get($0$1);
									}
								});
							}
							var div_4 = sibling(node_13, 2);
							CheckboxOptions(child(div_4), {
								save: "Params",
								get saveOn() {
									return get(productoForm);
								},
								type: "multiple",
								options: [{
									i: 1,
									v: "SKU Individual"
								}],
								keyId: "i",
								keyName: "v"
							});
							reset(div_4);
							var node_15 = sibling(div_4, 2);
							Input$1(node_15, {
								get saveOn() {
									return get(productoForm);
								},
								save: "Descripcion",
								css: "col-span-24 mb-4",
								label: "Descripción Corta"
							});
							var node_16 = sibling(node_15, 2);
							SearchCard(node_16, {
								css: "col-span-24 flex items-start",
								label: "CATEGORÍAS ::",
								get options() {
									return get(categorias);
								},
								keyId: "ID",
								keyName: "Nombre",
								cardCss: "grow",
								inputCss: "w-[35%] md:w-180",
								save: "CategoriasIDs",
								optionsCss: "w-280",
								get saveOn() {
									return get(productoForm);
								},
								set saveOn($$value) {
									set(productoForm, $$value, true);
								}
							});
							var node_17 = sibling(node_16, 4);
							Input$1(node_17, {
								get saveOn() {
									return get(productoForm);
								},
								save: "SbnUnidad",
								css: "col-span-12 md:col-span-6",
								label: "Nombre"
							});
							var node_18 = sibling(node_17, 2);
							Input$1(node_18, {
								save: "SbnPrecio",
								baseDecimals: 2,
								css: "col-span-12 md:col-span-6",
								label: "Precio Base",
								type: "number",
								onChange: () => {
									console.log();
									get(productoForm).PrecioFinal = Math.round(get(productoForm).Precio * (100 - (get(productoForm).Descuento || 0)) / 100);
									set(productoForm, { ...get(productoForm) }, true);
								},
								get saveOn() {
									return get(productoForm);
								},
								set saveOn($$value) {
									set(productoForm, $$value, true);
								}
							});
							var node_19 = sibling(node_18, 2);
							Input$1(node_19, {
								get saveOn() {
									return get(productoForm);
								},
								save: "SbnDescuento",
								postValue: "%",
								css: "col-span-12 md:col-span-6",
								label: "Descuento",
								type: "number"
							});
							var node_20 = sibling(node_19, 2);
							Input$1(node_20, {
								get saveOn() {
									return get(productoForm);
								},
								save: "SbnPreciFinal",
								baseDecimals: 2,
								css: "col-span-12 md:col-span-6",
								label: "Precio Final",
								type: "number"
							});
							Input$1(sibling(node_20, 2), {
								get saveOn() {
									return get(productoForm);
								},
								save: "SbnCantidad",
								css: "col-span-12 md:col-span-6",
								label: "Cantidad",
								type: "number"
							});
							reset(div_2);
							append($$anchor$3, div_2);
						};
						if_block(node_3, ($$render) => {
							if (get(layerView) === 1) $$render(consequent_1);
						});
						var node_22 = sibling(node_3, 2);
						var consequent_2 = ($$anchor$3) => {
							HTMLEditor($$anchor$3, {
								get saveOn() {
									return get(productoForm);
								},
								save: "ContentHTML",
								css: "mt-12"
							});
						};
						if_block(node_22, ($$render) => {
							if (get(layerView) === 2) $$render(consequent_2);
						});
						var node_23 = sibling(node_22, 2);
						var consequent_3 = ($$anchor$3) => {
							var div_5 = root_7();
							Atributos(child(div_5), {
								get producto() {
									return get(productoForm);
								},
								set producto($$value) {
									set(productoForm, $$value, true);
								}
							});
							reset(div_5);
							append($$anchor$3, div_5);
						};
						if_block(node_23, ($$render) => {
							if (get(layerView) === 3) $$render(consequent_3);
						});
						var node_25 = sibling(node_23, 2);
						var consequent_4 = ($$anchor$3) => {
							var div_6 = root_8();
							var node_26 = child(div_6);
							ImageUploader(node_26, {
								saveAPI: "producto-image",
								useConvertAvif: true,
								clearOnUpload: true,
								types: ["avif", "webp"],
								folder: "img-productos",
								cardCss: "w-full h-170 p-4",
								setDataToSend: (e$1) => {
									e$1.ProductoID = get(productoForm).ID;
								},
								onUploaded: (imagePath, description) => {
									if (imagePath.includes("/")) imagePath = imagePath.split("/")[1];
									get(productoForm).Image = {
										n: imagePath,
										d: description
									};
									get(productoForm).Images = get(productoForm).Images || [];
									get(productoForm).Images.unshift(get(productoForm).Image);
								}
							});
							each(sibling(node_26, 2), 17, () => get(productoForm).Images || [], index, ($$anchor$4, image) => {
								{
									let $0$1 = user_derived(() => get(image)?.n);
									ImageUploader($$anchor$4, {
										saveAPI: "producto-image",
										size: 2,
										clearOnUpload: true,
										types: ["avif", "webp"],
										folder: "img-productos",
										cardCss: "w-full h-170 p-4",
										get src() {
											return get($0$1);
										},
										useConvertAvif: true,
										onDelete: () => {
											ConfirmWarn("ELIMINAR IMAGEN", `Eliminar la imagen ${get(image).d ? `"${get(image).d}"` : "seleccionada"}`, "SI", "NO", () => {
												deleteProductoImage(get(image).n);
											});
										}
									});
								}
							});
							reset(div_6);
							append($$anchor$3, div_6);
						};
						if_block(node_25, ($$render) => {
							if (get(layerView) === 4) $$render(consequent_4);
						});
						append($$anchor$2, fragment_4);
					},
					$$slots: { default: true }
				});
			}
			var node_28 = sibling(node_2, 2);
			var consequent_5 = ($$anchor$2) => {
				bind_this(CategoriasMarcas($$anchor$2, {
					get listas() {
						return listas;
					},
					origin: 1
				}), ($$value) => CategoriasLayer = $$value, () => CategoriasLayer);
			};
			if_block(node_28, ($$render) => {
				if (get(view) === 2) $$render(consequent_5);
			});
			var node_29 = sibling(node_28, 2);
			var consequent_6 = ($$anchor$2) => {
				bind_this(CategoriasMarcas($$anchor$2, {
					get listas() {
						return listas;
					},
					origin: 2
				}), ($$value) => MarcasLayer = $$value, () => MarcasLayer);
			};
			if_block(node_29, ($$render) => {
				if (get(view) === 3) $$render(consequent_6);
			});
			append($$anchor$1, fragment_1);
		},
		$$slots: { default: true }
	});
	pop();
}
delegate(["keyup", "click"]);
export { _page as component };
