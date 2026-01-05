import { Tt as __toESM, xt as __commonJSMin } from "./CTB4HzdN.js";
var require_notiflix_aio_3_2_8_min = /* @__PURE__ */ __commonJSMin(((exports, module) => {
	(function(t, e) {
		"function" == typeof define && define.amd ? define([], function() {
			return e(t);
		}) : "object" == typeof module && "object" == typeof module.exports ? module.exports = e(t) : t.Notiflix = e(t);
	})("undefined" == typeof global ? "undefined" == typeof window ? exports : window : global, function(t) {
		"use strict";
		if ("undefined" == typeof t && "undefined" == typeof t.document) return !1;
		var e, i, a, n, o, r = "\n\nVisit documentation page to learn more: https://notiflix.github.io/documentation", s = "-apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, \"Helvetica Neue\", Arial, \"Noto Sans\", sans-serif", l = {
			Success: "Success",
			Failure: "Failure",
			Warning: "Warning",
			Info: "Info"
		}, m = {
			wrapID: "NotiflixNotifyWrap",
			overlayID: "NotiflixNotifyOverlay",
			width: "280px",
			position: "right-top",
			distance: "10px",
			opacity: 1,
			borderRadius: "5px",
			rtl: !1,
			timeout: 3e3,
			messageMaxLength: 110,
			backOverlay: !1,
			backOverlayColor: "rgba(0,0,0,0.5)",
			plainText: !0,
			showOnlyTheLastOne: !1,
			clickToClose: !1,
			pauseOnHover: !0,
			ID: "NotiflixNotify",
			className: "notiflix-notify",
			zindex: 4001,
			fontFamily: "Quicksand",
			fontSize: "13px",
			cssAnimation: !0,
			cssAnimationDuration: 400,
			cssAnimationStyle: "fade",
			closeButton: !1,
			useIcon: !0,
			useFontAwesome: !1,
			fontAwesomeIconStyle: "basic",
			fontAwesomeIconSize: "34px",
			success: {
				background: "#32c682",
				textColor: "#fff",
				childClassName: "notiflix-notify-success",
				notiflixIconColor: "rgba(0,0,0,0.2)",
				fontAwesomeClassName: "fas fa-check-circle",
				fontAwesomeIconColor: "rgba(0,0,0,0.2)",
				backOverlayColor: "rgba(50,198,130,0.2)"
			},
			failure: {
				background: "#ff5549",
				textColor: "#fff",
				childClassName: "notiflix-notify-failure",
				notiflixIconColor: "rgba(0,0,0,0.2)",
				fontAwesomeClassName: "fas fa-times-circle",
				fontAwesomeIconColor: "rgba(0,0,0,0.2)",
				backOverlayColor: "rgba(255,85,73,0.2)"
			},
			warning: {
				background: "#eebf31",
				textColor: "#fff",
				childClassName: "notiflix-notify-warning",
				notiflixIconColor: "rgba(0,0,0,0.2)",
				fontAwesomeClassName: "fas fa-exclamation-circle",
				fontAwesomeIconColor: "rgba(0,0,0,0.2)",
				backOverlayColor: "rgba(238,191,49,0.2)"
			},
			info: {
				background: "#26c0d3",
				textColor: "#fff",
				childClassName: "notiflix-notify-info",
				notiflixIconColor: "rgba(0,0,0,0.2)",
				fontAwesomeClassName: "fas fa-info-circle",
				fontAwesomeIconColor: "rgba(0,0,0,0.2)",
				backOverlayColor: "rgba(38,192,211,0.2)"
			}
		}, c = {
			Success: "Success",
			Failure: "Failure",
			Warning: "Warning",
			Info: "Info"
		}, p = {
			ID: "NotiflixReportWrap",
			className: "notiflix-report",
			width: "320px",
			backgroundColor: "#f8f8f8",
			borderRadius: "25px",
			rtl: !1,
			zindex: 4002,
			backOverlay: !0,
			backOverlayColor: "rgba(0,0,0,0.5)",
			backOverlayClickToClose: !1,
			fontFamily: "Quicksand",
			svgSize: "110px",
			plainText: !0,
			titleFontSize: "16px",
			titleMaxLength: 34,
			messageFontSize: "13px",
			messageMaxLength: 400,
			buttonFontSize: "14px",
			buttonMaxLength: 34,
			cssAnimation: !0,
			cssAnimationDuration: 360,
			cssAnimationStyle: "fade",
			success: {
				svgColor: "#32c682",
				titleColor: "#1e1e1e",
				messageColor: "#242424",
				buttonBackground: "#32c682",
				buttonColor: "#fff",
				backOverlayColor: "rgba(50,198,130,0.2)"
			},
			failure: {
				svgColor: "#ff5549",
				titleColor: "#1e1e1e",
				messageColor: "#242424",
				buttonBackground: "#ff5549",
				buttonColor: "#fff",
				backOverlayColor: "rgba(255,85,73,0.2)"
			},
			warning: {
				svgColor: "#eebf31",
				titleColor: "#1e1e1e",
				messageColor: "#242424",
				buttonBackground: "#eebf31",
				buttonColor: "#fff",
				backOverlayColor: "rgba(238,191,49,0.2)"
			},
			info: {
				svgColor: "#26c0d3",
				titleColor: "#1e1e1e",
				messageColor: "#242424",
				buttonBackground: "#26c0d3",
				buttonColor: "#fff",
				backOverlayColor: "rgba(38,192,211,0.2)"
			}
		}, f = {
			Show: "Show",
			Ask: "Ask",
			Prompt: "Prompt"
		}, d = {
			ID: "NotiflixConfirmWrap",
			className: "notiflix-confirm",
			width: "300px",
			zindex: 4003,
			position: "center",
			distance: "10px",
			backgroundColor: "#f8f8f8",
			borderRadius: "25px",
			backOverlay: !0,
			backOverlayColor: "rgba(0,0,0,0.5)",
			rtl: !1,
			fontFamily: "Quicksand",
			cssAnimation: !0,
			cssAnimationDuration: 300,
			cssAnimationStyle: "fade",
			plainText: !0,
			titleColor: "#32c682",
			titleFontSize: "16px",
			titleMaxLength: 34,
			messageColor: "#1e1e1e",
			messageFontSize: "14px",
			messageMaxLength: 110,
			buttonsFontSize: "15px",
			buttonsMaxLength: 34,
			okButtonColor: "#f8f8f8",
			okButtonBackground: "#32c682",
			cancelButtonColor: "#f8f8f8",
			cancelButtonBackground: "#a9a9a9"
		}, x = {
			Standard: "Standard",
			Hourglass: "Hourglass",
			Circle: "Circle",
			Arrows: "Arrows",
			Dots: "Dots",
			Pulse: "Pulse",
			Custom: "Custom",
			Notiflix: "Notiflix"
		}, g = {
			ID: "NotiflixLoadingWrap",
			className: "notiflix-loading",
			zindex: 4e3,
			backgroundColor: "rgba(0,0,0,0.8)",
			rtl: !1,
			fontFamily: "Quicksand",
			cssAnimation: !0,
			cssAnimationDuration: 400,
			clickToClose: !1,
			customSvgUrl: null,
			customSvgCode: null,
			svgSize: "80px",
			svgColor: "#32c682",
			messageID: "NotiflixLoadingMessage",
			messageFontSize: "15px",
			messageMaxLength: 34,
			messageColor: "#dcdcdc"
		}, b = {
			Standard: "Standard",
			Hourglass: "Hourglass",
			Circle: "Circle",
			Arrows: "Arrows",
			Dots: "Dots",
			Pulse: "Pulse"
		}, u = {
			ID: "NotiflixBlockWrap",
			querySelectorLimit: 200,
			className: "notiflix-block",
			position: "absolute",
			zindex: 1e3,
			backgroundColor: "rgba(255,255,255,0.9)",
			rtl: !1,
			fontFamily: "Quicksand",
			cssAnimation: !0,
			cssAnimationDuration: 300,
			svgSize: "45px",
			svgColor: "#383838",
			messageFontSize: "14px",
			messageMaxLength: 34,
			messageColor: "#383838"
		}, y = function(t$1) {
			return console.error("%c Notiflix Error ", "padding:2px;border-radius:20px;color:#fff;background:#ff5549", "\n" + t$1 + r);
		}, k = function(t$1) {
			return console.log("%c Notiflix Info ", "padding:2px;border-radius:20px;color:#fff;background:#26c0d3", "\n" + t$1 + r);
		}, w = function(e$1) {
			return e$1 || (e$1 = "head"), void 0 !== t.document[e$1] || (y("\nNotiflix needs to be appended to the \"<" + e$1 + ">\" element, but you called it before the \"<" + e$1 + ">\" element has been created."), !1);
		}, h = function(e$1, i$1) {
			if (!w("head")) return !1;
			if (null !== e$1() && !t.document.getElementById(i$1)) {
				var a$1 = t.document.createElement("style");
				a$1.id = i$1, a$1.innerHTML = e$1(), t.document.head.appendChild(a$1);
			}
		}, v = function() {
			var t$1 = {}, e$1 = !1, a$1 = 0;
			"[object Boolean]" === Object.prototype.toString.call(arguments[0]) && (e$1 = arguments[0], a$1++);
			for (var n$1 = function(i$1) {
				for (var a$2 in i$1) Object.prototype.hasOwnProperty.call(i$1, a$2) && (t$1[a$2] = e$1 && "[object Object]" === Object.prototype.toString.call(i$1[a$2]) ? v(t$1[a$2], i$1[a$2]) : i$1[a$2]);
			}; a$1 < arguments.length; a$1++) n$1(arguments[a$1]);
			return t$1;
		}, N = function(e$1) {
			var i$1 = t.document.createElement("div");
			return i$1.innerHTML = e$1, i$1.textContent || i$1.innerText || "";
		}, C = function(t$1, e$1) {
			t$1 || (t$1 = "110px"), e$1 || (e$1 = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXReportSuccess\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" fill=\"" + e$1 + "\" viewBox=\"0 0 120 120\"><style>@-webkit-keyframes NXReportSuccess1-animation{0%{-webkit-transform:translate(60px,57.7px) scale(.5,.5) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(.5,.5) translate(-60px,-57.7px)}50%,to{-webkit-transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px)}60%{-webkit-transform:translate(60px,57.7px) scale(.95,.95) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(.95,.95) translate(-60px,-57.7px)}}@keyframes NXReportSuccess1-animation{0%{-webkit-transform:translate(60px,57.7px) scale(.5,.5) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(.5,.5) translate(-60px,-57.7px)}50%,to{-webkit-transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px)}60%{-webkit-transform:translate(60px,57.7px) scale(.95,.95) translate(-60px,-57.7px);transform:translate(60px,57.7px) scale(.95,.95) translate(-60px,-57.7px)}}@-webkit-keyframes NXReportSuccess4-animation{0%{opacity:0}50%,to{opacity:1}}@keyframes NXReportSuccess4-animation{0%{opacity:0}50%,to{opacity:1}}@-webkit-keyframes NXReportSuccess3-animation{0%{opacity:0}40%,to{opacity:1}}@keyframes NXReportSuccess3-animation{0%{opacity:0}40%,to{opacity:1}}@-webkit-keyframes NXReportSuccess2-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportSuccess2-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}#NXReportSuccess *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g style=\"-webkit-animation-name:NXReportSuccess2-animation;animation-name:NXReportSuccess2-animation;-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\"><path d=\"M60 115.38C29.46 115.38 4.62 90.54 4.62 60 4.62 29.46 29.46 4.62 60 4.62c30.54 0 55.38 24.84 55.38 55.38 0 30.54-24.84 55.38-55.38 55.38zM60 0C26.92 0 0 26.92 0 60s26.92 60 60 60 60-26.92 60-60S93.08 0 60 0z\" style=\"-webkit-animation-name:NXReportSuccess3-animation;animation-name:NXReportSuccess3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g><g style=\"-webkit-animation-name:NXReportSuccess1-animation;animation-name:NXReportSuccess1-animation;-webkit-transform:translate(60px,57.7px) scale(1,1) translate(-60px,-57.7px);-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\"><path d=\"M88.27 35.39L52.8 75.29 31.43 58.2c-.98-.81-2.44-.63-3.24.36-.79.99-.63 2.44.36 3.24l23.08 18.46c.43.34.93.51 1.44.51.64 0 1.27-.26 1.74-.78l36.91-41.53a2.3 2.3 0 0 0-.19-3.26c-.95-.86-2.41-.77-3.26.19z\" style=\"-webkit-animation-name:NXReportSuccess4-animation;animation-name:NXReportSuccess4-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g></svg>";
		}, z = function(t$1, e$1) {
			t$1 || (t$1 = "110px"), e$1 || (e$1 = "#ff5549");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXReportFailure\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" fill=\"" + e$1 + "\" viewBox=\"0 0 120 120\"><style>@-webkit-keyframes NXReportFailure2-animation{0%{opacity:0}40%,to{opacity:1}}@keyframes NXReportFailure2-animation{0%{opacity:0}40%,to{opacity:1}}@-webkit-keyframes NXReportFailure1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportFailure1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@-webkit-keyframes NXReportFailure3-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}50%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportFailure3-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}50%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@-webkit-keyframes NXReportFailure4-animation{0%{opacity:0}50%,to{opacity:1}}@keyframes NXReportFailure4-animation{0%{opacity:0}50%,to{opacity:1}}#NXReportFailure *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g style=\"-webkit-animation-name:NXReportFailure1-animation;animation-name:NXReportFailure1-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M4.35 34.95c0-16.82 13.78-30.6 30.6-30.6h50.1c16.82 0 30.6 13.78 30.6 30.6v50.1c0 16.82-13.78 30.6-30.6 30.6h-50.1c-16.82 0-30.6-13.78-30.6-30.6v-50.1zM34.95 120h50.1c19.22 0 34.95-15.73 34.95-34.95v-50.1C120 15.73 104.27 0 85.05 0h-50.1C15.73 0 0 15.73 0 34.95v50.1C0 104.27 15.73 120 34.95 120z\" style=\"-webkit-animation-name:NXReportFailure2-animation;animation-name:NXReportFailure2-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g><g style=\"-webkit-animation-name:NXReportFailure3-animation;animation-name:NXReportFailure3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M82.4 37.6c-.9-.9-2.37-.9-3.27 0L60 56.73 40.86 37.6a2.306 2.306 0 0 0-3.26 3.26L56.73 60 37.6 79.13c-.9.9-.9 2.37 0 3.27.45.45 1.04.68 1.63.68.59 0 1.18-.23 1.63-.68L60 63.26 79.13 82.4c.45.45 1.05.68 1.64.68.58 0 1.18-.23 1.63-.68.9-.9.9-2.37 0-3.27L63.26 60 82.4 40.86c.9-.91.9-2.36 0-3.26z\" style=\"-webkit-animation-name:NXReportFailure4-animation;animation-name:NXReportFailure4-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g></svg>";
		}, S = function(t$1, e$1) {
			t$1 || (t$1 = "110px"), e$1 || (e$1 = "#eebf31");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXReportWarning\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" fill=\"" + e$1 + "\" viewBox=\"0 0 120 120\"><style>@-webkit-keyframes NXReportWarning2-animation{0%{opacity:0}40%,to{opacity:1}}@keyframes NXReportWarning2-animation{0%{opacity:0}40%,to{opacity:1}}@-webkit-keyframes NXReportWarning1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportWarning1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@-webkit-keyframes NXReportWarning3-animation{0%{-webkit-transform:translate(60px,66.6px) scale(.5,.5) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(.5,.5) translate(-60px,-66.6px)}50%,to{-webkit-transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px)}60%{-webkit-transform:translate(60px,66.6px) scale(.95,.95) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(.95,.95) translate(-60px,-66.6px)}}@keyframes NXReportWarning3-animation{0%{-webkit-transform:translate(60px,66.6px) scale(.5,.5) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(.5,.5) translate(-60px,-66.6px)}50%,to{-webkit-transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px)}60%{-webkit-transform:translate(60px,66.6px) scale(.95,.95) translate(-60px,-66.6px);transform:translate(60px,66.6px) scale(.95,.95) translate(-60px,-66.6px)}}@-webkit-keyframes NXReportWarning4-animation{0%{opacity:0}50%,to{opacity:1}}@keyframes NXReportWarning4-animation{0%{opacity:0}50%,to{opacity:1}}#NXReportWarning *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g style=\"-webkit-animation-name:NXReportWarning1-animation;animation-name:NXReportWarning1-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M115.46 106.15l-54.04-93.8c-.61-1.06-2.23-1.06-2.84 0l-54.04 93.8c-.62 1.07.21 2.29 1.42 2.29h108.08c1.21 0 2.04-1.22 1.42-2.29zM65.17 10.2l54.04 93.8c2.28 3.96-.65 8.78-5.17 8.78H5.96c-4.52 0-7.45-4.82-5.17-8.78l54.04-93.8c2.28-3.95 8.03-4 10.34 0z\" style=\"-webkit-animation-name:NXReportWarning2-animation;animation-name:NXReportWarning2-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g><g style=\"-webkit-animation-name:NXReportWarning3-animation;animation-name:NXReportWarning3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,66.6px) scale(1,1) translate(-60px,-66.6px)\"><path d=\"M57.83 94.01c0 1.2.97 2.17 2.17 2.17s2.17-.97 2.17-2.17v-3.2c0-1.2-.97-2.17-2.17-2.17s-2.17.97-2.17 2.17v3.2zm0-14.15c0 1.2.97 2.17 2.17 2.17s2.17-.97 2.17-2.17V39.21c0-1.2-.97-2.17-2.17-2.17s-2.17.97-2.17 2.17v40.65z\" style=\"-webkit-animation-name:NXReportWarning4-animation;animation-name:NXReportWarning4-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g></svg>";
		}, L = function(t$1, e$1) {
			t$1 || (t$1 = "110px"), e$1 || (e$1 = "#26c0d3");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXReportInfo\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" fill=\"" + e$1 + "\" viewBox=\"0 0 120 120\"><style>@-webkit-keyframes NXReportInfo4-animation{0%{opacity:0}50%,to{opacity:1}}@keyframes NXReportInfo4-animation{0%{opacity:0}50%,to{opacity:1}}@-webkit-keyframes NXReportInfo3-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}50%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportInfo3-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}50%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@-webkit-keyframes NXReportInfo2-animation{0%{opacity:0}40%,to{opacity:1}}@keyframes NXReportInfo2-animation{0%{opacity:0}40%,to{opacity:1}}@-webkit-keyframes NXReportInfo1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}@keyframes NXReportInfo1-animation{0%{-webkit-transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px);transform:translate(60px,60px) scale(.5,.5) translate(-60px,-60px)}40%,to{-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px);transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)}60%{-webkit-transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px);transform:translate(60px,60px) scale(.95,.95) translate(-60px,-60px)}}#NXReportInfo *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g style=\"-webkit-animation-name:NXReportInfo1-animation;animation-name:NXReportInfo1-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M60 115.38C29.46 115.38 4.62 90.54 4.62 60 4.62 29.46 29.46 4.62 60 4.62c30.54 0 55.38 24.84 55.38 55.38 0 30.54-24.84 55.38-55.38 55.38zM60 0C26.92 0 0 26.92 0 60s26.92 60 60 60 60-26.92 60-60S93.08 0 60 0z\" style=\"-webkit-animation-name:NXReportInfo2-animation;animation-name:NXReportInfo2-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g><g style=\"-webkit-animation-name:NXReportInfo3-animation;animation-name:NXReportInfo3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform:translate(60px,60px) scale(1,1) translate(-60px,-60px)\"><path d=\"M57.75 43.85c0-1.24 1.01-2.25 2.25-2.25s2.25 1.01 2.25 2.25v48.18c0 1.24-1.01 2.25-2.25 2.25s-2.25-1.01-2.25-2.25V43.85zm0-15.88c0-1.24 1.01-2.25 2.25-2.25s2.25 1.01 2.25 2.25v3.32c0 1.25-1.01 2.25-2.25 2.25s-2.25-1-2.25-2.25v-3.32z\" style=\"-webkit-animation-name:NXReportInfo4-animation;animation-name:NXReportInfo4-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1)\" fill=\"inherit\" data-animator-group=\"true\" data-animator-type=\"2\"/></g></svg>";
		}, W = function(t$1, e$1) {
			t$1 || (t$1 = "60px"), e$1 || (e$1 = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" stroke=\"" + e$1 + "\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" transform=\"scale(.8)\" viewBox=\"0 0 38 38\"><g fill=\"none\" fill-rule=\"evenodd\" stroke-width=\"2\" transform=\"translate(1 1)\"><circle cx=\"18\" cy=\"18\" r=\"18\" stroke-opacity=\".25\"/><path d=\"M36 18c0-9.94-8.06-18-18-18\"><animateTransform attributeName=\"transform\" dur=\"1s\" from=\"0 18 18\" repeatCount=\"indefinite\" to=\"360 18 18\" type=\"rotate\"/></path></g></svg>";
		}, I = function(t$1, e$1) {
			t$1 || (t$1 = "60px"), e$1 || (e$1 = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXLoadingHourglass\" fill=\"" + e$1 + "\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" viewBox=\"0 0 200 200\"><style>@-webkit-keyframes NXhourglass5-animation{0%{-webkit-transform:scale(1,1);transform:scale(1,1)}16.67%{-webkit-transform:scale(1,.8);transform:scale(1,.8)}33.33%{-webkit-transform:scale(.88,.6);transform:scale(.88,.6)}37.5%{-webkit-transform:scale(.85,.55);transform:scale(.85,.55)}41.67%{-webkit-transform:scale(.8,.5);transform:scale(.8,.5)}45.83%{-webkit-transform:scale(.75,.45);transform:scale(.75,.45)}50%{-webkit-transform:scale(.7,.4);transform:scale(.7,.4)}54.17%{-webkit-transform:scale(.6,.35);transform:scale(.6,.35)}58.33%{-webkit-transform:scale(.5,.3);transform:scale(.5,.3)}83.33%,to{-webkit-transform:scale(.2,0);transform:scale(.2,0)}}@keyframes NXhourglass5-animation{0%{-webkit-transform:scale(1,1);transform:scale(1,1)}16.67%{-webkit-transform:scale(1,.8);transform:scale(1,.8)}33.33%{-webkit-transform:scale(.88,.6);transform:scale(.88,.6)}37.5%{-webkit-transform:scale(.85,.55);transform:scale(.85,.55)}41.67%{-webkit-transform:scale(.8,.5);transform:scale(.8,.5)}45.83%{-webkit-transform:scale(.75,.45);transform:scale(.75,.45)}50%{-webkit-transform:scale(.7,.4);transform:scale(.7,.4)}54.17%{-webkit-transform:scale(.6,.35);transform:scale(.6,.35)}58.33%{-webkit-transform:scale(.5,.3);transform:scale(.5,.3)}83.33%,to{-webkit-transform:scale(.2,0);transform:scale(.2,0)}}@-webkit-keyframes NXhourglass3-animation{0%{-webkit-transform:scale(1,.02);transform:scale(1,.02)}79.17%,to{-webkit-transform:scale(1,1);transform:scale(1,1)}}@keyframes NXhourglass3-animation{0%{-webkit-transform:scale(1,.02);transform:scale(1,.02)}79.17%,to{-webkit-transform:scale(1,1);transform:scale(1,1)}}@-webkit-keyframes NXhourglass1-animation{0%,83.33%{-webkit-transform:rotate(0deg);transform:rotate(0deg)}to{-webkit-transform:rotate(180deg);transform:rotate(180deg)}}@keyframes NXhourglass1-animation{0%,83.33%{-webkit-transform:rotate(0deg);transform:rotate(0deg)}to{-webkit-transform:rotate(180deg);transform:rotate(180deg)}}#NXLoadingHourglass *{-webkit-animation-duration:1.2s;animation-duration:1.2s;-webkit-animation-iteration-count:infinite;animation-iteration-count:infinite;-webkit-animation-timing-function:cubic-bezier(0,0,1,1);animation-timing-function:cubic-bezier(0,0,1,1)}</style><g data-animator-group=\"true\" data-animator-type=\"1\" style=\"-webkit-animation-name:NXhourglass1-animation;animation-name:NXhourglass1-animation;-webkit-transform-origin:50% 50%;transform-origin:50% 50%;transform-box:fill-box\"><g id=\"NXhourglass2\" fill=\"inherit\"><g data-animator-group=\"true\" data-animator-type=\"2\" style=\"-webkit-animation-name:NXhourglass3-animation;animation-name:NXhourglass3-animation;-webkit-animation-timing-function:cubic-bezier(.42,0,.58,1);animation-timing-function:cubic-bezier(.42,0,.58,1);-webkit-transform-origin:50% 100%;transform-origin:50% 100%;transform-box:fill-box\" opacity=\".4\"><path id=\"NXhourglass4\" d=\"M100 100l-34.38 32.08v31.14h68.76v-31.14z\"/></g><g data-animator-group=\"true\" data-animator-type=\"2\" style=\"-webkit-animation-name:NXhourglass5-animation;animation-name:NXhourglass5-animation;-webkit-transform-origin:50% 100%;transform-origin:50% 100%;transform-box:fill-box\" opacity=\".4\"><path id=\"NXhourglass6\" d=\"M100 100L65.62 67.92V36.78h68.76v31.14z\"/></g><path d=\"M51.14 38.89h8.33v14.93c0 15.1 8.29 28.99 23.34 39.1 1.88 1.25 3.04 3.97 3.04 7.08s-1.16 5.83-3.04 7.09c-15.05 10.1-23.34 23.99-23.34 39.09v14.93h-8.33a4.859 4.859 0 1 0 0 9.72h97.72a4.859 4.859 0 1 0 0-9.72h-8.33v-14.93c0-15.1-8.29-28.99-23.34-39.09-1.88-1.26-3.04-3.98-3.04-7.09s1.16-5.83 3.04-7.08c15.05-10.11 23.34-24 23.34-39.1V38.89h8.33a4.859 4.859 0 1 0 0-9.72H51.14a4.859 4.859 0 1 0 0 9.72zm79.67 14.93c0 15.87-11.93 26.25-19.04 31.03-4.6 3.08-7.34 8.75-7.34 15.15 0 6.41 2.74 12.07 7.34 15.15 7.11 4.78 19.04 15.16 19.04 31.03v14.93H69.19v-14.93c0-15.87 11.93-26.25 19.04-31.02 4.6-3.09 7.34-8.75 7.34-15.16 0-6.4-2.74-12.07-7.34-15.15-7.11-4.78-19.04-15.16-19.04-31.03V38.89h61.62v14.93z\"/></g></g></svg>";
		}, R = function(t$1, e$1) {
			t$1 || (t$1 = "60px"), e$1 || (e$1 = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" viewBox=\"25 25 50 50\" style=\"-webkit-animation:rotate 2s linear infinite;animation:rotate 2s linear infinite;height:" + t$1 + ";-webkit-transform-origin:center center;-ms-transform-origin:center center;transform-origin:center center;width:" + t$1 + ";position:absolute;top:0;left:0;margin:auto\"><style>@-webkit-keyframes rotate{to{-webkit-transform:rotate(360deg);transform:rotate(360deg)}}@keyframes rotate{to{-webkit-transform:rotate(360deg);transform:rotate(360deg)}}@-webkit-keyframes dash{0%{stroke-dasharray:1,200;stroke-dashoffset:0}50%{stroke-dasharray:89,200;stroke-dashoffset:-35}to{stroke-dasharray:89,200;stroke-dashoffset:-124}}@keyframes dash{0%{stroke-dasharray:1,200;stroke-dashoffset:0}50%{stroke-dasharray:89,200;stroke-dashoffset:-35}to{stroke-dasharray:89,200;stroke-dashoffset:-124}}</style><circle cx=\"50\" cy=\"50\" r=\"20\" fill=\"none\" stroke=\"" + e$1 + "\" stroke-width=\"2\" style=\"-webkit-animation:dash 1.5s ease-in-out infinite,color 1.5s ease-in-out infinite;animation:dash 1.5s ease-in-out infinite,color 1.5s ease-in-out infinite\" stroke-dasharray=\"150 200\" stroke-dashoffset=\"-10\" stroke-linecap=\"round\"/></svg>";
		}, A = function(t$1, e$1) {
			t$1 || (t$1 = "60px"), e$1 || (e$1 = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" fill=\"" + e$1 + "\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" viewBox=\"0 0 128 128\"><g><path fill=\"inherit\" d=\"M109.25 55.5h-36l12-12a29.54 29.54 0 0 0-49.53 12H18.75A46.04 46.04 0 0 1 96.9 31.84l12.35-12.34v36zm-90.5 17h36l-12 12a29.54 29.54 0 0 0 49.53-12h16.97A46.04 46.04 0 0 1 31.1 96.16L18.74 108.5v-36z\"/><animateTransform attributeName=\"transform\" dur=\"1.5s\" from=\"0 64 64\" repeatCount=\"indefinite\" to=\"360 64 64\" type=\"rotate\"/></g></svg>";
		}, M = function(t$1, e$1) {
			t$1 || (t$1 = "60px"), e$1 || (e$1 = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" fill=\"" + e$1 + "\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" viewBox=\"0 0 100 100\"><g transform=\"translate(25 50)\"><circle r=\"9\" fill=\"inherit\" transform=\"scale(.239)\"><animateTransform attributeName=\"transform\" begin=\"-0.266s\" calcMode=\"spline\" dur=\"0.8s\" keySplines=\"0.3 0 0.7 1;0.3 0 0.7 1\" keyTimes=\"0;0.5;1\" repeatCount=\"indefinite\" type=\"scale\" values=\"0;1;0\"/></circle></g><g transform=\"translate(50 50)\"><circle r=\"9\" fill=\"inherit\" transform=\"scale(.00152)\"><animateTransform attributeName=\"transform\" begin=\"-0.133s\" calcMode=\"spline\" dur=\"0.8s\" keySplines=\"0.3 0 0.7 1;0.3 0 0.7 1\" keyTimes=\"0;0.5;1\" repeatCount=\"indefinite\" type=\"scale\" values=\"0;1;0\"/></circle></g><g transform=\"translate(75 50)\"><circle r=\"9\" fill=\"inherit\" transform=\"scale(.299)\"><animateTransform attributeName=\"transform\" begin=\"0s\" calcMode=\"spline\" dur=\"0.8s\" keySplines=\"0.3 0 0.7 1;0.3 0 0.7 1\" keyTimes=\"0;0.5;1\" repeatCount=\"indefinite\" type=\"scale\" values=\"0;1;0\"/></circle></g></svg>";
		}, B = function(t$1, e$1) {
			t$1 || (t$1 = "60px"), e$1 || (e$1 = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" stroke=\"" + e$1 + "\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" viewBox=\"0 0 44 44\"><g fill=\"none\" fill-rule=\"evenodd\" stroke-width=\"2\"><circle cx=\"22\" cy=\"22\" r=\"1\"><animate attributeName=\"r\" begin=\"0s\" calcMode=\"spline\" dur=\"1.8s\" keySplines=\"0.165, 0.84, 0.44, 1\" keyTimes=\"0; 1\" repeatCount=\"indefinite\" values=\"1; 20\"/><animate attributeName=\"stroke-opacity\" begin=\"0s\" calcMode=\"spline\" dur=\"1.8s\" keySplines=\"0.3, 0.61, 0.355, 1\" keyTimes=\"0; 1\" repeatCount=\"indefinite\" values=\"1; 0\"/></circle><circle cx=\"22\" cy=\"22\" r=\"1\"><animate attributeName=\"r\" begin=\"-0.9s\" calcMode=\"spline\" dur=\"1.8s\" keySplines=\"0.165, 0.84, 0.44, 1\" keyTimes=\"0; 1\" repeatCount=\"indefinite\" values=\"1; 20\"/><animate attributeName=\"stroke-opacity\" begin=\"-0.9s\" calcMode=\"spline\" dur=\"1.8s\" keySplines=\"0.3, 0.61, 0.355, 1\" keyTimes=\"0; 1\" repeatCount=\"indefinite\" values=\"1; 0\"/></circle></g></svg>";
		}, X = function(t$1, e$1, i$1) {
			t$1 || (t$1 = "60px"), e$1 || (e$1 = "#f8f8f8"), i$1 || (i$1 = "#32c682");
			return "<svg xmlns=\"http://www.w3.org/2000/svg\" id=\"NXLoadingNotiflixLib\" width=\"" + t$1 + "\" height=\"" + t$1 + "\" viewBox=\"0 0 200 200\"><defs><style>@keyframes notiflix-n{0%{stroke-dashoffset:1000}to{stroke-dashoffset:0}}@keyframes notiflix-x{0%{stroke-dashoffset:1000}to{stroke-dashoffset:0}}@keyframes notiflix-dot{0%,to{stroke-width:0}50%{stroke-width:12}}.nx-icon-line{stroke:" + e$1 + ";stroke-width:12;stroke-linecap:round;stroke-linejoin:round;stroke-miterlimit:22;fill:none}</style></defs><path d=\"M47.97 135.05a6.5 6.5 0 1 1 0 13 6.5 6.5 0 0 1 0-13z\" style=\"animation-name:notiflix-dot;animation-timing-function:ease-in-out;animation-duration:1.25s;animation-iteration-count:infinite;animation-direction:normal\" fill=\"" + i$1 + "\" stroke=\"" + i$1 + "\" stroke-linecap=\"round\" stroke-linejoin=\"round\" stroke-miterlimit=\"22\" stroke-width=\"12\"/><path class=\"nx-icon-line\" d=\"M10.14 144.76V87.55c0-5.68-4.54-41.36 37.83-41.36 42.36 0 37.82 35.68 37.82 41.36v57.21\" style=\"animation-name:notiflix-n;animation-timing-function:linear;animation-duration:2.5s;animation-delay:0s;animation-iteration-count:infinite;animation-direction:normal\" stroke-dasharray=\"500\"/><path class=\"nx-icon-line\" d=\"M115.06 144.49c24.98-32.68 49.96-65.35 74.94-98.03M114.89 46.6c25.09 32.58 50.19 65.17 75.29 97.75\" style=\"animation-name:notiflix-x;animation-timing-function:linear;animation-duration:2.5s;animation-delay:.2s;animation-iteration-count:infinite;animation-direction:normal\" stroke-dasharray=\"500\"/></svg>";
		}, D = function() {
			return "[id^=NotiflixNotifyWrap]{pointer-events:none;position:fixed;z-index:4001;opacity:1;right:10px;top:10px;width:280px;max-width:96%;-webkit-box-sizing:border-box;box-sizing:border-box;background:transparent}[id^=NotiflixNotifyWrap].nx-flex-center-center{max-height:calc(100vh - 20px);overflow-x:hidden;overflow-y:auto;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;margin:auto}[id^=NotiflixNotifyWrap]::-webkit-scrollbar{width:0;height:0}[id^=NotiflixNotifyWrap]::-webkit-scrollbar-thumb{background:transparent}[id^=NotiflixNotifyWrap]::-webkit-scrollbar-track{background:transparent}[id^=NotiflixNotifyWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixNotifyOverlay]{-webkit-transition:background .3s ease-in-out;-o-transition:background .3s ease-in-out;transition:background .3s ease-in-out}[id^=NotiflixNotifyWrap]>div{pointer-events:all;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif;width:100%;display:-webkit-inline-box;display:-webkit-inline-flex;display:-ms-inline-flexbox;display:inline-flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;position:relative;margin:0 0 10px;border-radius:5px;background:#1e1e1e;color:#fff;padding:10px 12px;font-size:14px;line-height:1.4}[id^=NotiflixNotifyWrap]>div:last-child{margin:0}[id^=NotiflixNotifyWrap]>div.nx-with-callback{cursor:pointer}[id^=NotiflixNotifyWrap]>div.nx-with-icon{padding:8px;min-height:56px}[id^=NotiflixNotifyWrap]>div.nx-paused{cursor:auto}[id^=NotiflixNotifyWrap]>div.nx-notify-click-to-close{cursor:pointer}[id^=NotiflixNotifyWrap]>div.nx-with-close-button{padding:10px 36px 10px 12px}[id^=NotiflixNotifyWrap]>div.nx-with-icon.nx-with-close-button{padding:6px 36px 6px 6px}[id^=NotiflixNotifyWrap]>div>span.nx-message{cursor:inherit;font-weight:normal;font-family:inherit!important;word-break:break-all;word-break:break-word}[id^=NotiflixNotifyWrap]>div>span.nx-close-button{cursor:pointer;-webkit-transition:all .2s ease-in-out;-o-transition:all .2s ease-in-out;transition:all .2s ease-in-out;position:absolute;right:8px;top:0;bottom:0;margin:auto;color:inherit;width:20px;height:20px}[id^=NotiflixNotifyWrap]>div>span.nx-close-button:hover{-webkit-transform:rotate(90deg);transform:rotate(90deg)}[id^=NotiflixNotifyWrap]>div>span.nx-close-button>svg{position:absolute;width:16px;height:16px;right:2px;top:2px}[id^=NotiflixNotifyWrap]>div>.nx-message-icon{position:absolute;width:40px;height:40px;font-size:30px;line-height:40px;text-align:center;left:8px;top:0;bottom:0;margin:auto;border-radius:inherit}[id^=NotiflixNotifyWrap]>div>.nx-message-icon-fa.nx-message-icon-fa-shadow{color:inherit;background:rgba(0,0,0,.15);-webkit-box-shadow:inset 0 0 34px rgba(0,0,0,.2);box-shadow:inset 0 0 34px rgba(0,0,0,.2);text-shadow:0 0 10px rgba(0,0,0,.3)}[id^=NotiflixNotifyWrap]>div>span.nx-with-icon{position:relative;float:left;width:calc(100% - 40px);margin:0 0 0 40px;padding:0 0 0 10px;-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixNotifyWrap]>div.nx-rtl-on>.nx-message-icon{left:auto;right:8px}[id^=NotiflixNotifyWrap]>div.nx-rtl-on>span.nx-with-icon{padding:0 10px 0 0;margin:0 40px 0 0}[id^=NotiflixNotifyWrap]>div.nx-rtl-on>span.nx-close-button{right:auto;left:8px}[id^=NotiflixNotifyWrap]>div.nx-with-icon.nx-with-close-button.nx-rtl-on{padding:6px 6px 6px 36px}[id^=NotiflixNotifyWrap]>div.nx-with-close-button.nx-rtl-on{padding:10px 12px 10px 36px}[id^=NotiflixNotifyOverlay].nx-with-animation,[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-fade{-webkit-animation:notify-animation-fade .3s ease-in-out 0s normal;animation:notify-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes notify-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-zoom{-webkit-animation:notify-animation-zoom .3s ease-in-out 0s normal;animation:notify-animation-zoom .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-zoom{0%{-webkit-transform:scale(0);transform:scale(0)}50%{-webkit-transform:scale(1.05);transform:scale(1.05)}100%{-webkit-transform:scale(1);transform:scale(1)}}@keyframes notify-animation-zoom{0%{-webkit-transform:scale(0);transform:scale(0)}50%{-webkit-transform:scale(1.05);transform:scale(1.05)}100%{-webkit-transform:scale(1);transform:scale(1)}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-right{-webkit-animation:notify-animation-from-right .3s ease-in-out 0s normal;animation:notify-animation-from-right .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-from-right{0%{right:-300px;opacity:0}50%{right:8px;opacity:1}100%{right:0;opacity:1}}@keyframes notify-animation-from-right{0%{right:-300px;opacity:0}50%{right:8px;opacity:1}100%{right:0;opacity:1}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-left{-webkit-animation:notify-animation-from-left .3s ease-in-out 0s normal;animation:notify-animation-from-left .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-from-left{0%{left:-300px;opacity:0}50%{left:8px;opacity:1}100%{left:0;opacity:1}}@keyframes notify-animation-from-left{0%{left:-300px;opacity:0}50%{left:8px;opacity:1}100%{left:0;opacity:1}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-top{-webkit-animation:notify-animation-from-top .3s ease-in-out 0s normal;animation:notify-animation-from-top .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-from-top{0%{top:-50px;opacity:0}50%{top:8px;opacity:1}100%{top:0;opacity:1}}@keyframes notify-animation-from-top{0%{top:-50px;opacity:0}50%{top:8px;opacity:1}100%{top:0;opacity:1}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-bottom{-webkit-animation:notify-animation-from-bottom .3s ease-in-out 0s normal;animation:notify-animation-from-bottom .3s ease-in-out 0s normal}@-webkit-keyframes notify-animation-from-bottom{0%{bottom:-50px;opacity:0}50%{bottom:8px;opacity:1}100%{bottom:0;opacity:1}}@keyframes notify-animation-from-bottom{0%{bottom:-50px;opacity:0}50%{bottom:8px;opacity:1}100%{bottom:0;opacity:1}}[id^=NotiflixNotifyOverlay].nx-with-animation.nx-remove,[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-fade.nx-remove{opacity:0;-webkit-animation:notify-remove-fade .3s ease-in-out 0s normal;animation:notify-remove-fade .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-fade{0%{opacity:1}100%{opacity:0}}@keyframes notify-remove-fade{0%{opacity:1}100%{opacity:0}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-zoom.nx-remove{-webkit-transform:scale(0);transform:scale(0);-webkit-animation:notify-remove-zoom .3s ease-in-out 0s normal;animation:notify-remove-zoom .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-zoom{0%{-webkit-transform:scale(1);transform:scale(1)}50%{-webkit-transform:scale(1.05);transform:scale(1.05)}100%{-webkit-transform:scale(0);transform:scale(0)}}@keyframes notify-remove-zoom{0%{-webkit-transform:scale(1);transform:scale(1)}50%{-webkit-transform:scale(1.05);transform:scale(1.05)}100%{-webkit-transform:scale(0);transform:scale(0)}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-top.nx-remove{opacity:0;-webkit-animation:notify-remove-to-top .3s ease-in-out 0s normal;animation:notify-remove-to-top .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-to-top{0%{top:0;opacity:1}50%{top:8px;opacity:1}100%{top:-50px;opacity:0}}@keyframes notify-remove-to-top{0%{top:0;opacity:1}50%{top:8px;opacity:1}100%{top:-50px;opacity:0}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-right.nx-remove{opacity:0;-webkit-animation:notify-remove-to-right .3s ease-in-out 0s normal;animation:notify-remove-to-right .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-to-right{0%{right:0;opacity:1}50%{right:8px;opacity:1}100%{right:-300px;opacity:0}}@keyframes notify-remove-to-right{0%{right:0;opacity:1}50%{right:8px;opacity:1}100%{right:-300px;opacity:0}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-bottom.nx-remove{opacity:0;-webkit-animation:notify-remove-to-bottom .3s ease-in-out 0s normal;animation:notify-remove-to-bottom .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-to-bottom{0%{bottom:0;opacity:1}50%{bottom:8px;opacity:1}100%{bottom:-50px;opacity:0}}@keyframes notify-remove-to-bottom{0%{bottom:0;opacity:1}50%{bottom:8px;opacity:1}100%{bottom:-50px;opacity:0}}[id^=NotiflixNotifyWrap]>div.nx-with-animation.nx-from-left.nx-remove{opacity:0;-webkit-animation:notify-remove-to-left .3s ease-in-out 0s normal;animation:notify-remove-to-left .3s ease-in-out 0s normal}@-webkit-keyframes notify-remove-to-left{0%{left:0;opacity:1}50%{left:8px;opacity:1}100%{left:-300px;opacity:0}}@keyframes notify-remove-to-left{0%{left:0;opacity:1}50%{left:8px;opacity:1}100%{left:-300px;opacity:0}}";
		}, T = 0, F = function(a$1, n$1, o$1, r$1) {
			if (!w("body")) return !1;
			e || G.Notify.init({});
			var c$1 = v(!0, e, {});
			if ("object" == typeof o$1 && !Array.isArray(o$1) || "object" == typeof r$1 && !Array.isArray(r$1)) {
				var p$1 = {};
				"object" == typeof o$1 ? p$1 = o$1 : "object" == typeof r$1 && (p$1 = r$1), e = v(!0, e, p$1);
			}
			var f$1 = e[a$1.toLocaleLowerCase("en")];
			T++, "string" != typeof n$1 && (n$1 = "Notiflix " + a$1), e.plainText && (n$1 = N(n$1)), !e.plainText && n$1.length > e.messageMaxLength && (e = v(!0, e, {
				closeButton: !0,
				messageMaxLength: 150
			}), n$1 = "Possible HTML Tags Error: The \"plainText\" option is \"false\" and the notification content length is more than the \"messageMaxLength\" option."), n$1.length > e.messageMaxLength && (n$1 = n$1.substring(0, e.messageMaxLength) + "..."), "shadow" === e.fontAwesomeIconStyle && (f$1.fontAwesomeIconColor = f$1.background), e.cssAnimation || (e.cssAnimationDuration = 0);
			var d$1 = t.document.getElementById(m.wrapID) || t.document.createElement("div");
			if (d$1.id = m.wrapID, d$1.style.width = e.width, d$1.style.zIndex = e.zindex, d$1.style.opacity = e.opacity, "center-center" === e.position ? (d$1.style.left = e.distance, d$1.style.top = e.distance, d$1.style.right = e.distance, d$1.style.bottom = e.distance, d$1.style.margin = "auto", d$1.classList.add("nx-flex-center-center"), d$1.style.maxHeight = "calc((100vh - " + e.distance + ") - " + e.distance + ")", d$1.style.display = "flex", d$1.style.flexWrap = "wrap", d$1.style.flexDirection = "column", d$1.style.justifyContent = "center", d$1.style.alignItems = "center", d$1.style.pointerEvents = "none") : "center-top" === e.position ? (d$1.style.left = e.distance, d$1.style.right = e.distance, d$1.style.top = e.distance, d$1.style.bottom = "auto", d$1.style.margin = "auto") : "center-bottom" === e.position ? (d$1.style.left = e.distance, d$1.style.right = e.distance, d$1.style.bottom = e.distance, d$1.style.top = "auto", d$1.style.margin = "auto") : "right-bottom" === e.position ? (d$1.style.right = e.distance, d$1.style.bottom = e.distance, d$1.style.top = "auto", d$1.style.left = "auto") : "left-top" === e.position ? (d$1.style.left = e.distance, d$1.style.top = e.distance, d$1.style.right = "auto", d$1.style.bottom = "auto") : "left-bottom" === e.position ? (d$1.style.left = e.distance, d$1.style.bottom = e.distance, d$1.style.top = "auto", d$1.style.right = "auto") : (d$1.style.right = e.distance, d$1.style.top = e.distance, d$1.style.left = "auto", d$1.style.bottom = "auto"), e.backOverlay) {
				var x$1 = t.document.getElementById(m.overlayID) || t.document.createElement("div");
				x$1.id = m.overlayID, x$1.style.width = "100%", x$1.style.height = "100%", x$1.style.position = "fixed", x$1.style.zIndex = e.zindex - 1, x$1.style.left = 0, x$1.style.top = 0, x$1.style.right = 0, x$1.style.bottom = 0, x$1.style.background = f$1.backOverlayColor || e.backOverlayColor, x$1.className = e.cssAnimation ? "nx-with-animation" : "", x$1.style.animationDuration = e.cssAnimation ? e.cssAnimationDuration + "ms" : "", t.document.getElementById(m.overlayID) || t.document.body.appendChild(x$1);
			}
			t.document.getElementById(m.wrapID) || t.document.body.appendChild(d$1);
			var g$1 = t.document.createElement("div");
			g$1.id = e.ID + "-" + T, g$1.className = e.className + " " + f$1.childClassName + " " + (e.cssAnimation ? "nx-with-animation" : "") + " " + (e.useIcon ? "nx-with-icon" : "") + " nx-" + e.cssAnimationStyle + " " + (e.closeButton && "function" != typeof o$1 ? "nx-with-close-button" : "") + " " + ("function" == typeof o$1 ? "nx-with-callback" : "") + " " + (e.clickToClose ? "nx-notify-click-to-close" : ""), g$1.style.fontSize = e.fontSize, g$1.style.color = f$1.textColor, g$1.style.background = f$1.background, g$1.style.borderRadius = e.borderRadius, g$1.style.pointerEvents = "all", e.rtl && (g$1.setAttribute("dir", "rtl"), g$1.classList.add("nx-rtl-on")), g$1.style.fontFamily = "\"" + e.fontFamily + "\", " + s, e.cssAnimation && (g$1.style.animationDuration = e.cssAnimationDuration + "ms");
			var b$1 = "";
			if (e.closeButton && "function" != typeof o$1 && (b$1 = "<span class=\"nx-close-button\"><svg xmlns=\"http://www.w3.org/2000/svg\" width=\"20\" height=\"20\" viewBox=\"0 0 20 20\"><g><path fill=\"" + f$1.notiflixIconColor + "\" d=\"M0.38 2.19l7.8 7.81 -7.8 7.81c-0.51,0.5 -0.51,1.31 -0.01,1.81 0.25,0.25 0.57,0.38 0.91,0.38 0.34,0 0.67,-0.14 0.91,-0.38l7.81 -7.81 7.81 7.81c0.24,0.24 0.57,0.38 0.91,0.38 0.34,0 0.66,-0.14 0.9,-0.38 0.51,-0.5 0.51,-1.31 0,-1.81l-7.81 -7.81 7.81 -7.81c0.51,-0.5 0.51,-1.31 0,-1.82 -0.5,-0.5 -1.31,-0.5 -1.81,0l-7.81 7.81 -7.81 -7.81c-0.5,-0.5 -1.31,-0.5 -1.81,0 -0.51,0.51 -0.51,1.32 0,1.82z\"/></g></svg></span>"), !e.useIcon) g$1.innerHTML = "<span class=\"nx-message\">" + n$1 + "</span>" + (e.closeButton ? b$1 : "");
			else if (e.useFontAwesome) g$1.innerHTML = "<i style=\"color:" + f$1.fontAwesomeIconColor + "; font-size:" + e.fontAwesomeIconSize + ";\" class=\"nx-message-icon nx-message-icon-fa " + f$1.fontAwesomeClassName + " " + ("shadow" === e.fontAwesomeIconStyle ? "nx-message-icon-fa-shadow" : "nx-message-icon-fa-basic") + "\"></i><span class=\"nx-message nx-with-icon\">" + n$1 + "</span>" + (e.closeButton ? b$1 : "");
			else {
				var u$1 = "";
				a$1 === l.Success ? u$1 = "<svg class=\"nx-message-icon\" xmlns=\"http://www.w3.org/2000/svg\" width=\"40\" height=\"40\" viewBox=\"0 0 40 40\"><g><path fill=\"" + f$1.notiflixIconColor + "\" d=\"M20 0c11.03,0 20,8.97 20,20 0,11.03 -8.97,20 -20,20 -11.03,0 -20,-8.97 -20,-20 0,-11.03 8.97,-20 20,-20zm0 37.98c9.92,0 17.98,-8.06 17.98,-17.98 0,-9.92 -8.06,-17.98 -17.98,-17.98 -9.92,0 -17.98,8.06 -17.98,17.98 0,9.92 8.06,17.98 17.98,17.98zm-2.4 -13.29l11.52 -12.96c0.37,-0.41 1.01,-0.45 1.42,-0.08 0.42,0.37 0.46,1 0.09,1.42l-12.16 13.67c-0.19,0.22 -0.46,0.34 -0.75,0.34 -0.23,0 -0.45,-0.07 -0.63,-0.22l-7.6 -6.07c-0.43,-0.35 -0.5,-0.99 -0.16,-1.42 0.35,-0.43 0.99,-0.5 1.42,-0.16l6.85 5.48z\"/></g></svg>" : a$1 === l.Failure ? u$1 = "<svg class=\"nx-message-icon\" xmlns=\"http://www.w3.org/2000/svg\" width=\"40\" height=\"40\" viewBox=\"0 0 40 40\"><g><path fill=\"" + f$1.notiflixIconColor + "\" d=\"M20 0c11.03,0 20,8.97 20,20 0,11.03 -8.97,20 -20,20 -11.03,0 -20,-8.97 -20,-20 0,-11.03 8.97,-20 20,-20zm0 37.98c9.92,0 17.98,-8.06 17.98,-17.98 0,-9.92 -8.06,-17.98 -17.98,-17.98 -9.92,0 -17.98,8.06 -17.98,17.98 0,9.92 8.06,17.98 17.98,17.98zm1.42 -17.98l6.13 6.12c0.39,0.4 0.39,1.04 0,1.43 -0.19,0.19 -0.45,0.29 -0.71,0.29 -0.27,0 -0.53,-0.1 -0.72,-0.29l-6.12 -6.13 -6.13 6.13c-0.19,0.19 -0.44,0.29 -0.71,0.29 -0.27,0 -0.52,-0.1 -0.71,-0.29 -0.39,-0.39 -0.39,-1.03 0,-1.43l6.13 -6.12 -6.13 -6.13c-0.39,-0.39 -0.39,-1.03 0,-1.42 0.39,-0.39 1.03,-0.39 1.42,0l6.13 6.12 6.12 -6.12c0.4,-0.39 1.04,-0.39 1.43,0 0.39,0.39 0.39,1.03 0,1.42l-6.13 6.13z\"/></g></svg>" : a$1 === l.Warning ? u$1 = "<svg class=\"nx-message-icon\" xmlns=\"http://www.w3.org/2000/svg\" width=\"40\" height=\"40\" viewBox=\"0 0 40 40\"><g><path fill=\"" + f$1.notiflixIconColor + "\" d=\"M21.91 3.48l17.8 30.89c0.84,1.46 -0.23,3.25 -1.91,3.25l-35.6 0c-1.68,0 -2.75,-1.79 -1.91,-3.25l17.8 -30.89c0.85,-1.47 2.97,-1.47 3.82,0zm16.15 31.84l-17.8 -30.89c-0.11,-0.2 -0.41,-0.2 -0.52,0l-17.8 30.89c-0.12,0.2 0.05,0.4 0.26,0.4l35.6 0c0.21,0 0.38,-0.2 0.26,-0.4zm-19.01 -4.12l0 -1.05c0,-0.53 0.42,-0.95 0.95,-0.95 0.53,0 0.95,0.42 0.95,0.95l0 1.05c0,0.53 -0.42,0.95 -0.95,0.95 -0.53,0 -0.95,-0.42 -0.95,-0.95zm0 -4.66l0 -13.39c0,-0.52 0.42,-0.95 0.95,-0.95 0.53,0 0.95,0.43 0.95,0.95l0 13.39c0,0.53 -0.42,0.96 -0.95,0.96 -0.53,0 -0.95,-0.43 -0.95,-0.96z\"/></g></svg>" : a$1 === l.Info && (u$1 = "<svg class=\"nx-message-icon\" xmlns=\"http://www.w3.org/2000/svg\" width=\"40\" height=\"40\" viewBox=\"0 0 40 40\"><g><path fill=\"" + f$1.notiflixIconColor + "\" d=\"M20 0c11.03,0 20,8.97 20,20 0,11.03 -8.97,20 -20,20 -11.03,0 -20,-8.97 -20,-20 0,-11.03 8.97,-20 20,-20zm0 37.98c9.92,0 17.98,-8.06 17.98,-17.98 0,-9.92 -8.06,-17.98 -17.98,-17.98 -9.92,0 -17.98,8.06 -17.98,17.98 0,9.92 8.06,17.98 17.98,17.98zm-0.99 -23.3c0,-0.54 0.44,-0.98 0.99,-0.98 0.55,0 0.99,0.44 0.99,0.98l0 15.86c0,0.55 -0.44,0.99 -0.99,0.99 -0.55,0 -0.99,-0.44 -0.99,-0.99l0 -15.86zm0 -5.22c0,-0.55 0.44,-0.99 0.99,-0.99 0.55,0 0.99,0.44 0.99,0.99l0 1.09c0,0.54 -0.44,0.99 -0.99,0.99 -0.55,0 -0.99,-0.45 -0.99,-0.99l0 -1.09z\"/></g></svg>"), g$1.innerHTML = u$1 + "<span class=\"nx-message nx-with-icon\">" + n$1 + "</span>" + (e.closeButton ? b$1 : "");
			}
			if ("left-bottom" === e.position || "right-bottom" === e.position) {
				var y$1 = t.document.getElementById(m.wrapID);
				y$1.insertBefore(g$1, y$1.firstChild);
			} else t.document.getElementById(m.wrapID).appendChild(g$1);
			var k$1 = t.document.getElementById(g$1.id);
			if (k$1) {
				var h$1, C$1, z$1 = function() {
					k$1.classList.add("nx-remove");
					var e$1 = t.document.getElementById(m.overlayID);
					e$1 && 0 >= d$1.childElementCount && e$1.classList.add("nx-remove"), clearTimeout(h$1);
				}, S$1 = function() {
					if (k$1 && null !== k$1.parentNode && k$1.parentNode.removeChild(k$1), 0 >= d$1.childElementCount && null !== d$1.parentNode) {
						d$1.parentNode.removeChild(d$1);
						var e$1 = t.document.getElementById(m.overlayID);
						e$1 && null !== e$1.parentNode && e$1.parentNode.removeChild(e$1);
					}
					clearTimeout(C$1);
				};
				if (e.closeButton && "function" != typeof o$1) t.document.getElementById(g$1.id).querySelector("span.nx-close-button").addEventListener("click", function() {
					z$1();
					var t$1 = setTimeout(function() {
						S$1(), clearTimeout(t$1);
					}, e.cssAnimationDuration);
				});
				if (("function" == typeof o$1 || e.clickToClose) && k$1.addEventListener("click", function() {
					"function" == typeof o$1 && o$1(), z$1();
					var t$1 = setTimeout(function() {
						S$1(), clearTimeout(t$1);
					}, e.cssAnimationDuration);
				}), !e.closeButton && "function" != typeof o$1) {
					var W$1 = function() {
						h$1 = setTimeout(function() {
							z$1();
						}, e.timeout), C$1 = setTimeout(function() {
							S$1();
						}, e.timeout + e.cssAnimationDuration);
					};
					W$1(), e.pauseOnHover && (k$1.addEventListener("mouseenter", function() {
						k$1.classList.add("nx-paused"), clearTimeout(h$1), clearTimeout(C$1);
					}), k$1.addEventListener("mouseleave", function() {
						k$1.classList.remove("nx-paused"), W$1();
					}));
				}
			}
			if (e.showOnlyTheLastOne && 0 < T) for (var I$1, R$1 = t.document.querySelectorAll("[id^=" + e.ID + "-]:not([id=" + e.ID + "-" + T + "])"), A$1 = 0; A$1 < R$1.length; A$1++) I$1 = R$1[A$1], null !== I$1.parentNode && I$1.parentNode.removeChild(I$1);
			e = v(!0, e, c$1);
		}, E = function() {
			return "[id^=NotiflixReportWrap]{position:fixed;z-index:4002;width:100%;height:100%;-webkit-box-sizing:border-box;box-sizing:border-box;font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif;left:0;top:0;padding:10px;color:#1e1e1e;border-radius:25px;background:transparent;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center}[id^=NotiflixReportWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixReportWrap]>div[class*=\"-overlay\"]{width:100%;height:100%;left:0;top:0;background:rgba(255,255,255,.5);position:fixed;z-index:0}[id^=NotiflixReportWrap]>div.nx-report-click-to-close{cursor:pointer}[id^=NotiflixReportWrap]>div[class*=\"-content\"]{width:320px;max-width:100%;max-height:96vh;overflow-x:hidden;overflow-y:auto;border-radius:inherit;padding:10px;-webkit-filter:drop-shadow(0 0 5px rgba(0,0,0,0.05));filter:drop-shadow(0 0 5px rgba(0, 0, 0, .05));border:1px solid rgba(0,0,0,.03);background:#f8f8f8;position:relative;z-index:1}[id^=NotiflixReportWrap]>div[class*=\"-content\"]::-webkit-scrollbar{width:0;height:0}[id^=NotiflixReportWrap]>div[class*=\"-content\"]::-webkit-scrollbar-thumb{background:transparent}[id^=NotiflixReportWrap]>div[class*=\"-content\"]::-webkit-scrollbar-track{background:transparent}[id^=NotiflixReportWrap]>div[class*=\"-content\"]>div[class$=\"-icon\"]{-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;width:110px;height:110px;display:block;margin:6px auto 12px}[id^=NotiflixReportWrap]>div[class*=\"-content\"]>div[class$=\"-icon\"] svg{min-width:100%;max-width:100%;height:auto}[id^=NotiflixReportWrap]>*>h5{word-break:break-all;word-break:break-word;font-family:inherit!important;font-size:16px;font-weight:500;line-height:1.4;margin:0 0 10px;padding:0 0 10px;border-bottom:1px solid rgba(0,0,0,.1);float:left;width:100%;text-align:center}[id^=NotiflixReportWrap]>*>p{word-break:break-all;word-break:break-word;font-family:inherit!important;font-size:13px;line-height:1.4;font-weight:normal;float:left;width:100%;padding:0 10px;margin:0 0 10px}[id^=NotiflixReportWrap] a#NXReportButton{word-break:break-all;word-break:break-word;-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;font-family:inherit!important;-webkit-transition:all .25s ease-in-out;-o-transition:all .25s ease-in-out;transition:all .25s ease-in-out;cursor:pointer;float:right;padding:7px 17px;background:#32c682;font-size:14px;line-height:1.4;font-weight:500;border-radius:inherit!important;color:#fff}[id^=NotiflixReportWrap] a#NXReportButton:hover{-webkit-box-shadow:inset 0 -60px 5px -5px rgba(0,0,0,.25);box-shadow:inset 0 -60px 5px -5px rgba(0,0,0,.25)}[id^=NotiflixReportWrap].nx-rtl-on a#NXReportButton{float:left}[id^=NotiflixReportWrap]>div[class*=\"-overlay\"].nx-with-animation{-webkit-animation:report-overlay-animation .3s ease-in-out 0s normal;animation:report-overlay-animation .3s ease-in-out 0s normal}@-webkit-keyframes report-overlay-animation{0%{opacity:0}100%{opacity:1}}@keyframes report-overlay-animation{0%{opacity:0}100%{opacity:1}}[id^=NotiflixReportWrap]>div[class*=\"-content\"].nx-with-animation.nx-fade{-webkit-animation:report-animation-fade .3s ease-in-out 0s normal;animation:report-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes report-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes report-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixReportWrap]>div[class*=\"-content\"].nx-with-animation.nx-zoom{-webkit-animation:report-animation-zoom .3s ease-in-out 0s normal;animation:report-animation-zoom .3s ease-in-out 0s normal}@-webkit-keyframes report-animation-zoom{0%{opacity:0;-webkit-transform:scale(.5);transform:scale(.5)}50%{opacity:1;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}}@keyframes report-animation-zoom{0%{opacity:0;-webkit-transform:scale(.5);transform:scale(.5)}50%{opacity:1;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}}[id^=NotiflixReportWrap].nx-remove>div[class*=\"-overlay\"].nx-with-animation{opacity:0;-webkit-animation:report-overlay-animation-remove .3s ease-in-out 0s normal;animation:report-overlay-animation-remove .3s ease-in-out 0s normal}@-webkit-keyframes report-overlay-animation-remove{0%{opacity:1}100%{opacity:0}}@keyframes report-overlay-animation-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixReportWrap].nx-remove>div[class*=\"-content\"].nx-with-animation.nx-fade{opacity:0;-webkit-animation:report-animation-fade-remove .3s ease-in-out 0s normal;animation:report-animation-fade-remove .3s ease-in-out 0s normal}@-webkit-keyframes report-animation-fade-remove{0%{opacity:1}100%{opacity:0}}@keyframes report-animation-fade-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixReportWrap].nx-remove>div[class*=\"-content\"].nx-with-animation.nx-zoom{opacity:0;-webkit-animation:report-animation-zoom-remove .3s ease-in-out 0s normal;animation:report-animation-zoom-remove .3s ease-in-out 0s normal}@-webkit-keyframes report-animation-zoom-remove{0%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}50%{opacity:.5;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:0;-webkit-transform:scale(0);transform:scale(0)}}@keyframes report-animation-zoom-remove{0%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}50%{opacity:.5;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:0;-webkit-transform:scale(0);transform:scale(0)}}";
		}, j = function(e$1, a$1, n$1, o$1, r$1, l$1) {
			if (!w("body")) return !1;
			i || G.Report.init({});
			var m$1 = {};
			if ("object" == typeof r$1 && !Array.isArray(r$1) || "object" == typeof l$1 && !Array.isArray(l$1)) {
				var f$1 = {};
				"object" == typeof r$1 ? f$1 = r$1 : "object" == typeof l$1 && (f$1 = l$1), m$1 = v(!0, i, {}), i = v(!0, i, f$1);
			}
			var d$1 = i[e$1.toLocaleLowerCase("en")];
			"string" != typeof a$1 && (a$1 = "Notiflix " + e$1), "string" != typeof n$1 && (e$1 === c.Success ? n$1 = "\"Do not try to become a person of success but try to become a person of value.\" <br><br>- Albert Einstein" : e$1 === c.Failure ? n$1 = "\"Failure is simply the opportunity to begin again, this time more intelligently.\" <br><br>- Henry Ford" : e$1 === c.Warning ? n$1 = "\"The peoples who want to live comfortably without producing and fatigue; they are doomed to lose their dignity, then liberty, and then independence and destiny.\" <br><br>- Mustafa Kemal Ataturk" : e$1 === c.Info && (n$1 = "\"Knowledge rests not upon truth alone, but upon error also.\" <br><br>- Carl Gustav Jung")), "string" != typeof o$1 && (o$1 = "Okay"), i.plainText && (a$1 = N(a$1), n$1 = N(n$1), o$1 = N(o$1)), i.plainText || (a$1.length > i.titleMaxLength && (a$1 = "Possible HTML Tags Error", n$1 = "The \"plainText\" option is \"false\" and the title content length is more than the \"titleMaxLength\" option.", o$1 = "Okay"), n$1.length > i.messageMaxLength && (a$1 = "Possible HTML Tags Error", n$1 = "The \"plainText\" option is \"false\" and the message content length is more than the \"messageMaxLength\" option.", o$1 = "Okay"), o$1.length > i.buttonMaxLength && (a$1 = "Possible HTML Tags Error", n$1 = "The \"plainText\" option is \"false\" and the button content length is more than the \"buttonMaxLength\" option.", o$1 = "Okay")), a$1.length > i.titleMaxLength && (a$1 = a$1.substring(0, i.titleMaxLength) + "..."), n$1.length > i.messageMaxLength && (n$1 = n$1.substring(0, i.messageMaxLength) + "..."), o$1.length > i.buttonMaxLength && (o$1 = o$1.substring(0, i.buttonMaxLength) + "..."), i.cssAnimation || (i.cssAnimationDuration = 0);
			var x$1 = t.document.createElement("div");
			x$1.id = p.ID, x$1.className = i.className, x$1.style.zIndex = i.zindex, x$1.style.borderRadius = i.borderRadius, x$1.style.fontFamily = "\"" + i.fontFamily + "\", " + s, i.rtl && (x$1.setAttribute("dir", "rtl"), x$1.classList.add("nx-rtl-on")), x$1.style.display = "flex", x$1.style.flexWrap = "wrap", x$1.style.flexDirection = "column", x$1.style.alignItems = "center", x$1.style.justifyContent = "center";
			var g$1 = "", b$1 = !0 === i.backOverlayClickToClose;
			i.backOverlay && (g$1 = "<div class=\"" + i.className + "-overlay" + (i.cssAnimation ? " nx-with-animation" : "") + (b$1 ? " nx-report-click-to-close" : "") + "\" style=\"background:" + (d$1.backOverlayColor || i.backOverlayColor) + ";animation-duration:" + i.cssAnimationDuration + "ms;\"></div>");
			var u$1 = "";
			if (e$1 === c.Success ? u$1 = C(i.svgSize, d$1.svgColor) : e$1 === c.Failure ? u$1 = z(i.svgSize, d$1.svgColor) : e$1 === c.Warning ? u$1 = S(i.svgSize, d$1.svgColor) : e$1 === c.Info && (u$1 = L(i.svgSize, d$1.svgColor)), x$1.innerHTML = g$1 + "<div class=\"" + i.className + "-content" + (i.cssAnimation ? " nx-with-animation " : "") + " nx-" + i.cssAnimationStyle + "\" style=\"width:" + i.width + "; background:" + i.backgroundColor + "; animation-duration:" + i.cssAnimationDuration + "ms;\"><div style=\"width:" + i.svgSize + "; height:" + i.svgSize + ";\" class=\"" + i.className + "-icon\">" + u$1 + "</div><h5 class=\"" + i.className + "-title\" style=\"font-weight:500; font-size:" + i.titleFontSize + "; color:" + d$1.titleColor + ";\">" + a$1 + "</h5><p class=\"" + i.className + "-message\" style=\"font-size:" + i.messageFontSize + "; color:" + d$1.messageColor + ";\">" + n$1 + "</p><a id=\"NXReportButton\" class=\"" + i.className + "-button\" style=\"font-weight:500; font-size:" + i.buttonFontSize + "; background:" + d$1.buttonBackground + "; color:" + d$1.buttonColor + ";\">" + o$1 + "</a></div>", !t.document.getElementById(x$1.id)) {
				t.document.body.appendChild(x$1);
				var y$1 = function() {
					var e$2 = t.document.getElementById(x$1.id);
					e$2.classList.add("nx-remove");
					var a$2 = setTimeout(function() {
						null !== e$2.parentNode && e$2.parentNode.removeChild(e$2), clearTimeout(a$2);
					}, i.cssAnimationDuration);
				};
				if (t.document.getElementById("NXReportButton").addEventListener("click", function() {
					"function" == typeof r$1 && r$1(), y$1();
				}), g$1 && b$1) t.document.querySelector(".nx-report-click-to-close").addEventListener("click", function() {
					y$1();
				});
			}
			i = v(!0, i, m$1);
		}, O = function() {
			return "[id^=NotiflixConfirmWrap]{position:fixed;z-index:4003;width:100%;height:100%;left:0;top:0;padding:10px;-webkit-box-sizing:border-box;box-sizing:border-box;background:transparent;font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center}[id^=NotiflixConfirmWrap].nx-position-center-top{-webkit-box-pack:start;-webkit-justify-content:flex-start;-ms-flex-pack:start;justify-content:flex-start}[id^=NotiflixConfirmWrap].nx-position-center-bottom{-webkit-box-pack:end;-webkit-justify-content:flex-end;-ms-flex-pack:end;justify-content:flex-end}[id^=NotiflixConfirmWrap].nx-position-left-top{-webkit-box-align:start;-webkit-align-items:flex-start;-ms-flex-align:start;align-items:flex-start;-webkit-box-pack:start;-webkit-justify-content:flex-start;-ms-flex-pack:start;justify-content:flex-start}[id^=NotiflixConfirmWrap].nx-position-left-center{-webkit-box-align:start;-webkit-align-items:flex-start;-ms-flex-align:start;align-items:flex-start}[id^=NotiflixConfirmWrap].nx-position-left-bottom{-webkit-box-align:start;-webkit-align-items:flex-start;-ms-flex-align:start;align-items:flex-start;-webkit-box-pack:end;-webkit-justify-content:flex-end;-ms-flex-pack:end;justify-content:flex-end}[id^=NotiflixConfirmWrap].nx-position-right-top{-webkit-box-align:end;-webkit-align-items:flex-end;-ms-flex-align:end;align-items:flex-end;-webkit-box-pack:start;-webkit-justify-content:flex-start;-ms-flex-pack:start;justify-content:flex-start}[id^=NotiflixConfirmWrap].nx-position-right-center{-webkit-box-align:end;-webkit-align-items:flex-end;-ms-flex-align:end;align-items:flex-end}[id^=NotiflixConfirmWrap].nx-position-right-bottom{-webkit-box-align:end;-webkit-align-items:flex-end;-ms-flex-align:end;align-items:flex-end;-webkit-box-pack:end;-webkit-justify-content:flex-end;-ms-flex-pack:end;justify-content:flex-end}[id^=NotiflixConfirmWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixConfirmWrap]>div[class*=\"-overlay\"]{width:100%;height:100%;left:0;top:0;background:rgba(255,255,255,.5);position:fixed;z-index:0}[id^=NotiflixConfirmWrap]>div[class*=\"-overlay\"].nx-with-animation{-webkit-animation:confirm-overlay-animation .3s ease-in-out 0s normal;animation:confirm-overlay-animation .3s ease-in-out 0s normal}@-webkit-keyframes confirm-overlay-animation{0%{opacity:0}100%{opacity:1}}@keyframes confirm-overlay-animation{0%{opacity:0}100%{opacity:1}}[id^=NotiflixConfirmWrap].nx-remove>div[class*=\"-overlay\"].nx-with-animation{opacity:0;-webkit-animation:confirm-overlay-animation-remove .3s ease-in-out 0s normal;animation:confirm-overlay-animation-remove .3s ease-in-out 0s normal}@-webkit-keyframes confirm-overlay-animation-remove{0%{opacity:1}100%{opacity:0}}@keyframes confirm-overlay-animation-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]{width:300px;max-width:100%;max-height:96vh;overflow-x:hidden;overflow-y:auto;border-radius:25px;padding:10px;margin:0;-webkit-filter:drop-shadow(0 0 5px rgba(0,0,0,0.05));filter:drop-shadow(0 0 5px rgba(0, 0, 0, .05));background:#f8f8f8;color:#1e1e1e;position:relative;z-index:1;text-align:center}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]::-webkit-scrollbar{width:0;height:0}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]::-webkit-scrollbar-thumb{background:transparent}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]::-webkit-scrollbar-track{background:transparent}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]{float:left;width:100%;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>h5{float:left;width:100%;margin:0;padding:0 0 10px;border-bottom:1px solid rgba(0,0,0,.1);color:#32c682;font-family:inherit!important;font-size:16px;line-height:1.4;font-weight:500;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div{font-family:inherit!important;margin:15px 0 20px;padding:0 10px;float:left;width:100%;font-size:14px;line-height:1.4;font-weight:normal;color:inherit;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div{font-family:inherit!important;float:left;width:100%;margin:15px 0 0;padding:0}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input{font-family:inherit!important;float:left;width:100%;height:40px;margin:0;padding:0 15px;border:1px solid rgba(0,0,0,.1);border-radius:25px;font-size:14px;font-weight:normal;line-height:1;-webkit-transition:all .25s ease-in-out;-o-transition:all .25s ease-in-out;transition:all .25s ease-in-out;text-align:left}[id^=NotiflixConfirmWrap].nx-rtl-on>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input{text-align:right}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input:hover{border-color:rgba(0,0,0,.1)}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input:focus{border-color:rgba(0,0,0,.3)}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input.nx-validation-failure{border-color:#ff5549}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-head\"]>div>div>input.nx-validation-success{border-color:#32c682}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]{-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;border-radius:inherit;float:left;width:100%;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a{cursor:pointer;font-family:inherit!important;-webkit-transition:all .25s ease-in-out;-o-transition:all .25s ease-in-out;transition:all .25s ease-in-out;float:left;width:48%;padding:9px 5px;border-radius:inherit!important;font-weight:500;font-size:15px;line-height:1.4;color:#f8f8f8;text-align:inherit}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a.nx-confirm-button-ok{margin:0 2% 0 0;background:#32c682}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a.nx-confirm-button-cancel{margin:0 0 0 2%;background:#a9a9a9}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a.nx-full{margin:0;width:100%}[id^=NotiflixConfirmWrap]>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a:hover{-webkit-box-shadow:inset 0 -60px 5px -5px rgba(0,0,0,.25);box-shadow:inset 0 -60px 5px -5px rgba(0,0,0,.25)}[id^=NotiflixConfirmWrap].nx-rtl-on>div[class*=\"-content\"]>div[class*=\"-buttons\"],[id^=NotiflixConfirmWrap].nx-rtl-on>div[class*=\"-content\"]>div[class*=\"-buttons\"]>a{-webkit-transform:rotateY(180deg);transform:rotateY(180deg)}[id^=NotiflixConfirmWrap].nx-with-animation.nx-fade>div[class*=\"-content\"]{-webkit-animation:confirm-animation-fade .3s ease-in-out 0s normal;animation:confirm-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes confirm-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes confirm-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixConfirmWrap].nx-with-animation.nx-zoom>div[class*=\"-content\"]{-webkit-animation:confirm-animation-zoom .3s ease-in-out 0s normal;animation:confirm-animation-zoom .3s ease-in-out 0s normal}@-webkit-keyframes confirm-animation-zoom{0%{opacity:0;-webkit-transform:scale(.5);transform:scale(.5)}50%{opacity:1;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}}@keyframes confirm-animation-zoom{0%{opacity:0;-webkit-transform:scale(.5);transform:scale(.5)}50%{opacity:1;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}}[id^=NotiflixConfirmWrap].nx-with-animation.nx-fade.nx-remove>div[class*=\"-content\"]{opacity:0;-webkit-animation:confirm-animation-fade-remove .3s ease-in-out 0s normal;animation:confirm-animation-fade-remove .3s ease-in-out 0s normal}@-webkit-keyframes confirm-animation-fade-remove{0%{opacity:1}100%{opacity:0}}@keyframes confirm-animation-fade-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixConfirmWrap].nx-with-animation.nx-zoom.nx-remove>div[class*=\"-content\"]{opacity:0;-webkit-animation:confirm-animation-zoom-remove .3s ease-in-out 0s normal;animation:confirm-animation-zoom-remove .3s ease-in-out 0s normal}@-webkit-keyframes confirm-animation-zoom-remove{0%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}50%{opacity:.5;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:0;-webkit-transform:scale(0);transform:scale(0)}}@keyframes confirm-animation-zoom-remove{0%{opacity:1;-webkit-transform:scale(1);transform:scale(1)}50%{opacity:.5;-webkit-transform:scale(1.05);transform:scale(1.05)}100%{opacity:0;-webkit-transform:scale(0);transform:scale(0)}}";
		}, H = function(e$1, i$1, n$1, o$1, r$1, l$1, m$1, c$1, p$1) {
			if (!w("body")) return !1;
			a || G.Confirm.init({});
			var x$1 = v(!0, a, {});
			"object" != typeof p$1 || Array.isArray(p$1) || (a = v(!0, a, p$1)), "string" != typeof i$1 && (i$1 = "Notiflix Confirm"), "string" != typeof n$1 && (n$1 = "Do you agree with me?"), "string" != typeof r$1 && (r$1 = "Yes"), "string" != typeof l$1 && (l$1 = "No"), "function" != typeof m$1 && (m$1 = void 0), "function" != typeof c$1 && (c$1 = void 0), a.plainText && (i$1 = N(i$1), n$1 = N(n$1), r$1 = N(r$1), l$1 = N(l$1)), a.plainText || (i$1.length > a.titleMaxLength && (i$1 = "Possible HTML Tags Error", n$1 = "The \"plainText\" option is \"false\" and the title content length is more than \"titleMaxLength\" option.", r$1 = "Okay", l$1 = "..."), n$1.length > a.messageMaxLength && (i$1 = "Possible HTML Tags Error", n$1 = "The \"plainText\" option is \"false\" and the message content length is more than \"messageMaxLength\" option.", r$1 = "Okay", l$1 = "..."), (r$1.length || l$1.length) > a.buttonsMaxLength && (i$1 = "Possible HTML Tags Error", n$1 = "The \"plainText\" option is \"false\" and the buttons content length is more than \"buttonsMaxLength\" option.", r$1 = "Okay", l$1 = "...")), i$1.length > a.titleMaxLength && (i$1 = i$1.substring(0, a.titleMaxLength) + "..."), n$1.length > a.messageMaxLength && (n$1 = n$1.substring(0, a.messageMaxLength) + "..."), r$1.length > a.buttonsMaxLength && (r$1 = r$1.substring(0, a.buttonsMaxLength) + "..."), l$1.length > a.buttonsMaxLength && (l$1 = l$1.substring(0, a.buttonsMaxLength) + "..."), a.cssAnimation || (a.cssAnimationDuration = 0);
			var g$1 = t.document.createElement("div");
			g$1.id = d.ID, g$1.className = a.className + (a.cssAnimation ? " nx-with-animation nx-" + a.cssAnimationStyle : ""), g$1.style.zIndex = a.zindex, g$1.style.padding = a.distance, a.rtl && (g$1.setAttribute("dir", "rtl"), g$1.classList.add("nx-rtl-on"));
			var b$1 = "string" == typeof a.position ? a.position.trim() : "center";
			g$1.classList.add("nx-position-" + b$1), g$1.style.fontFamily = "\"" + a.fontFamily + "\", " + s;
			var u$1 = "";
			a.backOverlay && (u$1 = "<div class=\"" + a.className + "-overlay" + (a.cssAnimation ? " nx-with-animation" : "") + "\" style=\"background:" + a.backOverlayColor + ";animation-duration:" + a.cssAnimationDuration + "ms;\"></div>");
			var y$1 = "";
			"function" == typeof m$1 && (y$1 = "<a id=\"NXConfirmButtonCancel\" class=\"nx-confirm-button-cancel\" style=\"color:" + a.cancelButtonColor + ";background:" + a.cancelButtonBackground + ";font-size:" + a.buttonsFontSize + ";\">" + l$1 + "</a>");
			var k$1 = "", h$1 = null, C$1 = void 0;
			if (e$1 === f.Ask || e$1 === f.Prompt) {
				h$1 = o$1 || "";
				var z$1 = e$1 === f.Ask ? Math.ceil(1.5 * h$1.length) : 200 < h$1.length ? Math.ceil(1.5 * h$1.length) : 250;
				k$1 = "<div><input id=\"NXConfirmValidationInput\" type=\"text\" " + (e$1 === f.Prompt ? "value=\"" + h$1 + "\"" : "") + " maxlength=\"" + z$1 + "\" style=\"font-size:" + a.messageFontSize + ";border-radius: " + a.borderRadius + ";\" autocomplete=\"off\" spellcheck=\"false\" autocapitalize=\"none\" /></div>";
			}
			if (g$1.innerHTML = u$1 + "<div class=\"" + a.className + "-content\" style=\"width:" + a.width + "; background:" + a.backgroundColor + "; animation-duration:" + a.cssAnimationDuration + "ms; border-radius: " + a.borderRadius + ";\"><div class=\"" + a.className + "-head\"><h5 style=\"color:" + a.titleColor + ";font-size:" + a.titleFontSize + ";\">" + i$1 + "</h5><div style=\"color:" + a.messageColor + ";font-size:" + a.messageFontSize + ";\">" + n$1 + k$1 + "</div></div><div class=\"" + a.className + "-buttons\"><a id=\"NXConfirmButtonOk\" class=\"nx-confirm-button-ok" + ("function" == typeof m$1 ? "" : " nx-full") + "\" style=\"color:" + a.okButtonColor + ";background:" + a.okButtonBackground + ";font-size:" + a.buttonsFontSize + ";\">" + r$1 + "</a>" + y$1 + "</div></div>", !t.document.getElementById(g$1.id)) {
				t.document.body.appendChild(g$1);
				var L$1 = t.document.getElementById(g$1.id), W$1 = t.document.getElementById("NXConfirmButtonOk"), I$1 = t.document.getElementById("NXConfirmValidationInput");
				if (I$1 && (I$1.focus(), I$1.setSelectionRange(0, (I$1.value || "").length), I$1.addEventListener("keyup", function(t$1) {
					var i$2 = t$1.target.value;
					if (e$1 === f.Ask && i$2 !== h$1) t$1.preventDefault(), I$1.classList.add("nx-validation-failure"), I$1.classList.remove("nx-validation-success");
					else {
						e$1 === f.Ask && (I$1.classList.remove("nx-validation-failure"), I$1.classList.add("nx-validation-success"));
						("enter" === (t$1.key || "").toLocaleLowerCase("en") || 13 === t$1.keyCode) && W$1.dispatchEvent(new Event("click"));
					}
				})), W$1.addEventListener("click", function(t$1) {
					if (e$1 === f.Ask && h$1 && I$1) {
						if ((I$1.value || "").toString() !== h$1) return I$1.focus(), I$1.classList.add("nx-validation-failure"), t$1.stopPropagation(), t$1.preventDefault(), t$1.returnValue = !1, t$1.cancelBubble = !0, !1;
						I$1.classList.remove("nx-validation-failure");
					}
					"function" == typeof m$1 && (e$1 === f.Prompt && I$1 && (C$1 = I$1.value || ""), m$1(C$1)), L$1.classList.add("nx-remove");
					var n$2 = setTimeout(function() {
						null !== L$1.parentNode && (L$1.parentNode.removeChild(L$1), clearTimeout(n$2));
					}, a.cssAnimationDuration);
				}), "function" == typeof m$1) t.document.getElementById("NXConfirmButtonCancel").addEventListener("click", function() {
					"function" == typeof c$1 && (e$1 === f.Prompt && I$1 && (C$1 = I$1.value || ""), c$1(C$1)), L$1.classList.add("nx-remove");
					var t$1 = setTimeout(function() {
						null !== L$1.parentNode && (L$1.parentNode.removeChild(L$1), clearTimeout(t$1));
					}, a.cssAnimationDuration);
				});
			}
			a = v(!0, a, x$1);
		}, P = function() {
			return "[id^=NotiflixLoadingWrap]{-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;position:fixed;z-index:4000;width:100%;height:100%;left:0;top:0;right:0;bottom:0;margin:auto;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center;text-align:center;-webkit-box-sizing:border-box;box-sizing:border-box;background:rgba(0,0,0,.8);font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif}[id^=NotiflixLoadingWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixLoadingWrap].nx-loading-click-to-close{cursor:pointer}[id^=NotiflixLoadingWrap]>div[class*=\"-icon\"]{width:60px;height:60px;position:relative;-webkit-transition:top .2s ease-in-out;-o-transition:top .2s ease-in-out;transition:top .2s ease-in-out;margin:0 auto}[id^=NotiflixLoadingWrap]>div[class*=\"-icon\"] img,[id^=NotiflixLoadingWrap]>div[class*=\"-icon\"] svg{max-width:unset;max-height:unset;width:100%;height:auto;position:absolute;left:0;top:0}[id^=NotiflixLoadingWrap]>p{position:relative;margin:10px auto 0;font-family:inherit!important;font-weight:normal;font-size:15px;line-height:1.4;padding:0 10px;width:100%;text-align:center}[id^=NotiflixLoadingWrap].nx-with-animation{-webkit-animation:loading-animation-fade .3s ease-in-out 0s normal;animation:loading-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes loading-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes loading-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixLoadingWrap].nx-with-animation.nx-remove{opacity:0;-webkit-animation:loading-animation-fade-remove .3s ease-in-out 0s normal;animation:loading-animation-fade-remove .3s ease-in-out 0s normal}@-webkit-keyframes loading-animation-fade-remove{0%{opacity:1}100%{opacity:0}}@keyframes loading-animation-fade-remove{0%{opacity:1}100%{opacity:0}}[id^=NotiflixLoadingWrap]>p.nx-loading-message-new{-webkit-animation:loading-new-message-fade .3s ease-in-out 0s normal;animation:loading-new-message-fade .3s ease-in-out 0s normal}@-webkit-keyframes loading-new-message-fade{0%{opacity:0}100%{opacity:1}}@keyframes loading-new-message-fade{0%{opacity:0}100%{opacity:1}}";
		}, U = function(e$1, i$1, a$1, o$1, r$1) {
			if (!w("body")) return !1;
			n || G.Loading.init({});
			var l$1 = v(!0, n, {});
			if ("object" == typeof i$1 && !Array.isArray(i$1) || "object" == typeof a$1 && !Array.isArray(a$1)) {
				var m$1 = {};
				"object" == typeof i$1 ? m$1 = i$1 : "object" == typeof a$1 && (m$1 = a$1), n = v(!0, n, m$1);
			}
			var c$1 = "";
			if ("string" == typeof i$1 && 0 < i$1.length && (c$1 = i$1), o$1) {
				c$1 = c$1.length > n.messageMaxLength ? N(c$1).toString().substring(0, n.messageMaxLength) + "..." : N(c$1).toString();
				var p$1 = "";
				0 < c$1.length && (p$1 = "<p id=\"" + n.messageID + "\" class=\"nx-loading-message\" style=\"color:" + n.messageColor + ";font-size:" + n.messageFontSize + ";\">" + c$1 + "</p>"), n.cssAnimation || (n.cssAnimationDuration = 0);
				var f$1 = "";
				if (e$1 === x.Standard) f$1 = W(n.svgSize, n.svgColor);
				else if (e$1 === x.Hourglass) f$1 = I(n.svgSize, n.svgColor);
				else if (e$1 === x.Circle) f$1 = R(n.svgSize, n.svgColor);
				else if (e$1 === x.Arrows) f$1 = A(n.svgSize, n.svgColor);
				else if (e$1 === x.Dots) f$1 = M(n.svgSize, n.svgColor);
				else if (e$1 === x.Pulse) f$1 = B(n.svgSize, n.svgColor);
				else if (e$1 === x.Custom && null !== n.customSvgCode && null === n.customSvgUrl) f$1 = n.customSvgCode || "";
				else if (e$1 === x.Custom && null !== n.customSvgUrl && null === n.customSvgCode) f$1 = "<img class=\"nx-custom-loading-icon\" width=\"" + n.svgSize + "\" height=\"" + n.svgSize + "\" src=\"" + n.customSvgUrl + "\" alt=\"Notiflix\">";
				else {
					if (e$1 === x.Custom && (null === n.customSvgUrl || null === n.customSvgCode)) return y("You have to set a static SVG url to \"customSvgUrl\" option to use Loading Custom."), !1;
					f$1 = X(n.svgSize, "#f8f8f8", "#32c682");
				}
				var d$1 = parseInt((n.svgSize || "").replace(/[^0-9]/g, "")), b$1 = t.innerWidth, u$1 = d$1 >= b$1 ? b$1 - 40 + "px" : d$1 + "px", k$1 = "<div style=\"width:" + u$1 + "; height:" + u$1 + ";\" class=\"" + n.className + "-icon" + (0 < c$1.length ? " nx-with-message" : "") + "\">" + f$1 + "</div>", h$1 = t.document.createElement("div");
				if (h$1.id = g.ID, h$1.className = n.className + (n.cssAnimation ? " nx-with-animation" : "") + (n.clickToClose ? " nx-loading-click-to-close" : ""), h$1.style.zIndex = n.zindex, h$1.style.background = n.backgroundColor, h$1.style.animationDuration = n.cssAnimationDuration + "ms", h$1.style.fontFamily = "\"" + n.fontFamily + "\", " + s, h$1.style.display = "flex", h$1.style.flexWrap = "wrap", h$1.style.flexDirection = "column", h$1.style.alignItems = "center", h$1.style.justifyContent = "center", n.rtl && (h$1.setAttribute("dir", "rtl"), h$1.classList.add("nx-rtl-on")), h$1.innerHTML = k$1 + p$1, !t.document.getElementById(h$1.id) && (t.document.body.appendChild(h$1), n.clickToClose)) t.document.getElementById(h$1.id).addEventListener("click", function() {
					h$1.classList.add("nx-remove");
					var t$1 = setTimeout(function() {
						null !== h$1.parentNode && (h$1.parentNode.removeChild(h$1), clearTimeout(t$1));
					}, n.cssAnimationDuration);
				});
			} else if (t.document.getElementById(g.ID)) var z$1 = t.document.getElementById(g.ID), S$1 = setTimeout(function() {
				z$1.classList.add("nx-remove");
				var t$1 = setTimeout(function() {
					null !== z$1.parentNode && (z$1.parentNode.removeChild(z$1), clearTimeout(t$1));
				}, n.cssAnimationDuration);
				clearTimeout(S$1);
			}, r$1);
			n = v(!0, n, l$1);
		}, V = function(e$1) {
			"string" != typeof e$1 && (e$1 = "");
			var i$1 = t.document.getElementById(g.ID);
			if (i$1) if (0 < e$1.length) {
				e$1 = e$1.length > n.messageMaxLength ? N(e$1).substring(0, n.messageMaxLength) + "..." : N(e$1);
				var a$1 = i$1.getElementsByTagName("p")[0];
				if (a$1) a$1.innerHTML = e$1;
				else {
					var o$1 = t.document.createElement("p");
					o$1.id = n.messageID, o$1.className = "nx-loading-message nx-loading-message-new", o$1.style.color = n.messageColor, o$1.style.fontSize = n.messageFontSize, o$1.innerHTML = e$1, i$1.appendChild(o$1);
				}
			} else y("Where is the new message?");
		}, q = function() {
			return "[id^=NotiflixBlockWrap]{-webkit-user-select:none;-moz-user-select:none;-ms-user-select:none;user-select:none;-webkit-box-sizing:border-box;box-sizing:border-box;position:absolute;z-index:1000;font-family:\"Quicksand\",-apple-system,BlinkMacSystemFont,\"Segoe UI\",Roboto,\"Helvetica Neue\",Arial,sans-serif;background:rgba(255,255,255,.9);text-align:center;animation-duration:.4s;width:100%;height:100%;left:0;top:0;border-radius:inherit;display:-webkit-box;display:-webkit-flex;display:-ms-flexbox;display:flex;-webkit-flex-wrap:wrap;-ms-flex-wrap:wrap;flex-wrap:wrap;-webkit-box-orient:vertical;-webkit-box-direction:normal;-webkit-flex-direction:column;-ms-flex-direction:column;flex-direction:column;-webkit-box-align:center;-webkit-align-items:center;-ms-flex-align:center;align-items:center;-webkit-box-pack:center;-webkit-justify-content:center;-ms-flex-pack:center;justify-content:center}[id^=NotiflixBlockWrap] *{-webkit-box-sizing:border-box;box-sizing:border-box}[id^=NotiflixBlockWrap]>span[class*=\"-icon\"]{display:block;width:45px;height:45px;position:relative;margin:0 auto}[id^=NotiflixBlockWrap]>span[class*=\"-icon\"] svg{width:inherit;height:inherit}[id^=NotiflixBlockWrap]>span[class*=\"-message\"]{position:relative;display:block;width:100%;margin:10px auto 0;padding:0 10px;font-family:inherit!important;font-weight:normal;font-size:14px;line-height:1.4}[id^=NotiflixBlockWrap].nx-with-animation{-webkit-animation:block-animation-fade .3s ease-in-out 0s normal;animation:block-animation-fade .3s ease-in-out 0s normal}@-webkit-keyframes block-animation-fade{0%{opacity:0}100%{opacity:1}}@keyframes block-animation-fade{0%{opacity:0}100%{opacity:1}}[id^=NotiflixBlockWrap].nx-with-animation.nx-remove{opacity:0;-webkit-animation:block-animation-fade-remove .3s ease-in-out 0s normal;animation:block-animation-fade-remove .3s ease-in-out 0s normal}@-webkit-keyframes block-animation-fade-remove{0%{opacity:1}100%{opacity:0}}@keyframes block-animation-fade-remove{0%{opacity:1}100%{opacity:0}}";
		}, Q = 0, Y = function(e$1, i$1, a$1, n$1, r$1, l$1) {
			var m$1;
			if (Array.isArray(a$1)) {
				if (1 > a$1.length) return y("Array of HTMLElements should contains at least one HTMLElement."), !1;
				m$1 = a$1;
			} else if (Object.prototype.isPrototypeOf.call(NodeList.prototype, a$1)) {
				if (1 > a$1.length) return y("NodeListOf<HTMLElement> should contains at least one HTMLElement."), !1;
				m$1 = Array.prototype.slice.call(a$1);
			} else {
				if ("string" != typeof a$1 || 1 > (a$1 || "").length || 1 === (a$1 || "").length && ("#" === (a$1 || "")[0] || "." === (a$1 || "")[0])) return y("The selector parameter must be a string and matches a specified CSS selector(s)."), !1;
				var p$1 = t.document.querySelectorAll(a$1);
				if (1 > p$1.length) return y("You called the \"Notiflix.Block...\" function with \"" + a$1 + "\" selector, but there is no such element(s) in the document."), !1;
				m$1 = p$1;
			}
			o || G.Block.init({});
			var f$1 = v(!0, o, {});
			if ("object" == typeof n$1 && !Array.isArray(n$1) || "object" == typeof r$1 && !Array.isArray(r$1)) {
				var d$1 = {};
				"object" == typeof n$1 ? d$1 = n$1 : "object" == typeof r$1 && (d$1 = r$1), o = v(!0, o, d$1);
			}
			var x$1 = "";
			"string" == typeof n$1 && 0 < n$1.length && (x$1 = n$1), o.cssAnimation || (o.cssAnimationDuration = 0);
			var g$1 = u.className;
			"string" == typeof o.className && (g$1 = o.className.trim());
			var h$1 = "number" == typeof o.querySelectorLimit ? o.querySelectorLimit : 200, C$1 = (m$1 || []).length >= h$1 ? h$1 : m$1.length, z$1 = "nx-block-temporary-position";
			if (e$1) {
				for (var S$1, L$1 = [
					"area",
					"base",
					"br",
					"col",
					"command",
					"embed",
					"hr",
					"img",
					"input",
					"keygen",
					"link",
					"meta",
					"param",
					"source",
					"track",
					"wbr",
					"html",
					"head",
					"title",
					"script",
					"style",
					"iframe"
				], X$1 = 0; X$1 < C$1; X$1++) if (S$1 = m$1[X$1], S$1) {
					if (-1 < L$1.indexOf(S$1.tagName.toLocaleLowerCase("en"))) break;
					var D$1 = S$1.querySelectorAll("[id^=" + u.ID + "]");
					if (1 > D$1.length) {
						var T$1 = "";
						i$1 && (i$1 === b.Hourglass ? T$1 = I(o.svgSize, o.svgColor) : i$1 === b.Circle ? T$1 = R(o.svgSize, o.svgColor) : i$1 === b.Arrows ? T$1 = A(o.svgSize, o.svgColor) : i$1 === b.Dots ? T$1 = M(o.svgSize, o.svgColor) : i$1 === b.Pulse ? T$1 = B(o.svgSize, o.svgColor) : T$1 = W(o.svgSize, o.svgColor));
						var F$1 = "<span class=\"" + g$1 + "-icon\" style=\"width:" + o.svgSize + ";height:" + o.svgSize + ";\">" + T$1 + "</span>", E$1 = "";
						0 < x$1.length && (x$1 = x$1.length > o.messageMaxLength ? N(x$1).substring(0, o.messageMaxLength) + "..." : N(x$1), E$1 = "<span style=\"font-size:" + o.messageFontSize + ";color:" + o.messageColor + ";\" class=\"" + g$1 + "-message\">" + x$1 + "</span>"), Q++;
						var j$1 = t.document.createElement("div");
						j$1.id = u.ID + "-" + Q, j$1.className = g$1 + (o.cssAnimation ? " nx-with-animation" : ""), j$1.style.position = o.position, j$1.style.zIndex = o.zindex, j$1.style.background = o.backgroundColor, j$1.style.animationDuration = o.cssAnimationDuration + "ms", j$1.style.fontFamily = "\"" + o.fontFamily + "\", " + s, j$1.style.display = "flex", j$1.style.flexWrap = "wrap", j$1.style.flexDirection = "column", j$1.style.alignItems = "center", j$1.style.justifyContent = "center", o.rtl && (j$1.setAttribute("dir", "rtl"), j$1.classList.add("nx-rtl-on")), j$1.innerHTML = F$1 + E$1;
						var O$1 = t.getComputedStyle(S$1).getPropertyValue("position"), H$1 = "string" == typeof O$1 ? O$1.toLocaleLowerCase("en") : "relative", P$1 = Math.round(1.25 * parseInt(o.svgSize)) + 40, U$1 = S$1.offsetHeight || 0, V$1 = "";
						P$1 > U$1 && (V$1 = "min-height:" + P$1 + "px;");
						var q$1 = "";
						q$1 = S$1.getAttribute("id") ? "#" + S$1.getAttribute("id") : S$1.classList[0] ? "." + S$1.classList[0] : (S$1.tagName || "").toLocaleLowerCase("en");
						var Y$1 = "", K = -1 >= [
							"absolute",
							"relative",
							"fixed",
							"sticky"
						].indexOf(H$1);
						if (K || 0 < V$1.length) {
							if (!w("head")) return !1;
							K && (Y$1 = "position:relative!important;");
							var $ = "<style id=\"Style-" + u.ID + "-" + Q + "\">" + q$1 + "." + z$1 + "{" + Y$1 + V$1 + "}</style>", J = t.document.createRange();
							J.selectNode(t.document.head);
							var Z = J.createContextualFragment($);
							t.document.head.appendChild(Z), S$1.classList.add(z$1);
						}
						S$1.appendChild(j$1);
					}
				}
			} else var _ = function(e$2) {
				var i$2 = setTimeout(function() {
					null !== e$2.parentNode && e$2.parentNode.removeChild(e$2);
					var a$2 = e$2.getAttribute("id"), n$2 = t.document.getElementById("Style-" + a$2);
					n$2 && null !== n$2.parentNode && n$2.parentNode.removeChild(n$2), clearTimeout(i$2);
				}, o.cssAnimationDuration);
			}, tt = function(t$1) {
				if (t$1 && 0 < t$1.length) for (var e$2, n$2 = 0; n$2 < t$1.length; n$2++) e$2 = t$1[n$2], e$2 && (e$2.classList.add("nx-remove"), _(e$2));
				else "string" == typeof a$1 ? k("\"Notiflix.Block.remove();\" function called with \"" + a$1 + "\" selector, but this selector does not have a \"Block\" element to remove.") : k("\"Notiflix.Block.remove();\" function called with \"" + a$1 + "\", but this \"Array<HTMLElement>\" or \"NodeListOf<HTMLElement>\" does not have a \"Block\" element to remove.");
			}, et = function(t$1) {
				var e$2 = setTimeout(function() {
					t$1.classList.remove(z$1), clearTimeout(e$2);
				}, o.cssAnimationDuration + 300);
			}, it = setTimeout(function() {
				for (var t$1, e$2 = 0; e$2 < C$1; e$2++) t$1 = m$1[e$2], t$1 && (et(t$1), D$1 = t$1.querySelectorAll("[id^=" + u.ID + "]"), tt(D$1));
				clearTimeout(it);
			}, l$1);
			o = v(!0, o, f$1);
		}, G = {
			Notify: {
				init: function(t$1) {
					e = v(!0, m, t$1), h(D, "NotiflixNotifyInternalCSS");
				},
				merge: function(t$1) {
					return e ? void (e = v(!0, e, t$1)) : (y("You have to initialize the Notify module before call Merge function."), !1);
				},
				success: function(t$1, e$1, i$1) {
					F(l.Success, t$1, e$1, i$1);
				},
				failure: function(t$1, e$1, i$1) {
					F(l.Failure, t$1, e$1, i$1);
				},
				warning: function(t$1, e$1, i$1) {
					F(l.Warning, t$1, e$1, i$1);
				},
				info: function(t$1, e$1, i$1) {
					F(l.Info, t$1, e$1, i$1);
				}
			},
			Report: {
				init: function(t$1) {
					i = v(!0, p, t$1), h(E, "NotiflixReportInternalCSS");
				},
				merge: function(t$1) {
					return i ? void (i = v(!0, i, t$1)) : (y("You have to initialize the Report module before call Merge function."), !1);
				},
				success: function(t$1, e$1, i$1, a$1, n$1) {
					j(c.Success, t$1, e$1, i$1, a$1, n$1);
				},
				failure: function(t$1, e$1, i$1, a$1, n$1) {
					j(c.Failure, t$1, e$1, i$1, a$1, n$1);
				},
				warning: function(t$1, e$1, i$1, a$1, n$1) {
					j(c.Warning, t$1, e$1, i$1, a$1, n$1);
				},
				info: function(t$1, e$1, i$1, a$1, n$1) {
					j(c.Info, t$1, e$1, i$1, a$1, n$1);
				}
			},
			Confirm: {
				init: function(t$1) {
					a = v(!0, d, t$1), h(O, "NotiflixConfirmInternalCSS");
				},
				merge: function(t$1) {
					return a ? void (a = v(!0, a, t$1)) : (y("You have to initialize the Confirm module before call Merge function."), !1);
				},
				show: function(t$1, e$1, i$1, a$1, n$1, o$1, r$1) {
					H(f.Show, t$1, e$1, null, i$1, a$1, n$1, o$1, r$1);
				},
				ask: function(t$1, e$1, i$1, a$1, n$1, o$1, r$1, s$1) {
					H(f.Ask, t$1, e$1, i$1, a$1, n$1, o$1, r$1, s$1);
				},
				prompt: function(t$1, e$1, i$1, a$1, n$1, o$1, r$1, s$1) {
					H(f.Prompt, t$1, e$1, i$1, a$1, n$1, o$1, r$1, s$1);
				}
			},
			Loading: {
				init: function(t$1) {
					n = v(!0, g, t$1), h(P, "NotiflixLoadingInternalCSS");
				},
				merge: function(t$1) {
					return n ? void (n = v(!0, n, t$1)) : (y("You have to initialize the Loading module before call Merge function."), !1);
				},
				standard: function(t$1, e$1) {
					U(x.Standard, t$1, e$1, !0, 0);
				},
				hourglass: function(t$1, e$1) {
					U(x.Hourglass, t$1, e$1, !0, 0);
				},
				circle: function(t$1, e$1) {
					U(x.Circle, t$1, e$1, !0, 0);
				},
				arrows: function(t$1, e$1) {
					U(x.Arrows, t$1, e$1, !0, 0);
				},
				dots: function(t$1, e$1) {
					U(x.Dots, t$1, e$1, !0, 0);
				},
				pulse: function(t$1, e$1) {
					U(x.Pulse, t$1, e$1, !0, 0);
				},
				custom: function(t$1, e$1) {
					U(x.Custom, t$1, e$1, !0, 0);
				},
				notiflix: function(t$1, e$1) {
					U(x.Notiflix, t$1, e$1, !0, 0);
				},
				remove: function(t$1) {
					"number" != typeof t$1 && (t$1 = 0), U(null, null, null, !1, t$1);
				},
				change: function(t$1) {
					V(t$1);
				}
			},
			Block: {
				init: function(t$1) {
					o = v(!0, u, t$1), h(q, "NotiflixBlockInternalCSS");
				},
				merge: function(t$1) {
					return o ? void (o = v(!0, o, t$1)) : (y("You have to initialize the \"Notiflix.Block\" module before call Merge function."), !1);
				},
				standard: function(t$1, e$1, i$1) {
					Y(!0, b.Standard, t$1, e$1, i$1);
				},
				hourglass: function(t$1, e$1, i$1) {
					Y(!0, b.Hourglass, t$1, e$1, i$1);
				},
				circle: function(t$1, e$1, i$1) {
					Y(!0, b.Circle, t$1, e$1, i$1);
				},
				arrows: function(t$1, e$1, i$1) {
					Y(!0, b.Arrows, t$1, e$1, i$1);
				},
				dots: function(t$1, e$1, i$1) {
					Y(!0, b.Dots, t$1, e$1, i$1);
				},
				pulse: function(t$1, e$1, i$1) {
					Y(!0, b.Pulse, t$1, e$1, i$1);
				},
				remove: function(t$1, e$1) {
					"number" != typeof e$1 && (e$1 = 0), Y(!1, null, t$1, null, null, e$1);
				}
			}
		};
		return "object" == typeof t.Notiflix ? v(!0, t.Notiflix, {
			Notify: G.Notify,
			Report: G.Report,
			Confirm: G.Confirm,
			Loading: G.Loading,
			Block: G.Block
		}) : {
			Notify: G.Notify,
			Report: G.Report,
			Confirm: G.Confirm,
			Loading: G.Loading,
			Block: G.Block
		};
	});
}));
const { Notify } = (/* @__PURE__ */ __toESM(require_notiflix_aio_3_2_8_min(), 1)).default;
var throttleTimer;
const throttle = (func, delay) => {
	if (throttleTimer) clearTimeout(throttleTimer);
	throttleTimer = setTimeout(() => {
		func();
		throttleTimer = null;
	}, delay);
};
const highlString = (phrase, words) => {
	if (typeof phrase !== "string") {
		console.error("no es string");
		console.log(phrase);
		return [{ text: "!" }];
	}
	const arr = [{ text: phrase }];
	if (!words || words.length === 0) return arr;
	console.log("words 222:", arr.filter((x) => x), "|", phrase, words);
	for (let word of words) {
		if (word.length < 2) continue;
		for (let i = 0; i < arr.length; i++) {
			const str = arr[i].text;
			if (typeof str !== "string") continue;
			const idx = str.toLowerCase().indexOf(word);
			if (idx !== -1) {
				const ini = str.slice(0, idx);
				const middle = str.slice(idx, idx + word.length);
				const fin = str.slice(idx + word.length);
				const parts = [
					{ text: ini },
					{
						text: middle,
						highl: true
					},
					{ text: fin }
				].filter((x) => x.text);
				arr.splice(i, 1, ...parts);
				if (arr.length > 40) {
					console.log("words 333:", arr.filter((x) => x), "|", phrase, words);
					return arr.filter((x) => x);
				}
				continue;
			}
		}
	}
	console.log("words 111:", arr.filter((x) => x), "|", phrase, words);
	return arr.filter((x) => x);
};
const parseSVG = (svgContent) => {
	return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svgContent)}`;
};
function include(e, h) {
	if (h && typeof h === "string") h = h.split(" ").filter((x) => x.length > 0);
	if (!h || h === "undefined" || h.length === 0) return true;
	else if (h.length === 1) return e.includes(h[0]);
	else if (h.length === 2) return e.includes(h[0]) && e.includes(h[1]);
	else if (h.length === 3) return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2]);
	else if (h.length === 4) return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2]) && e.includes(h[3]);
	else if (h.length === 5) return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2]) && e.includes(h[3]) && e.includes(h[4]);
	else return e.includes(h[0]) && e.includes(h[1]) && e.includes(h[2]) && e.includes(h[3]) && e.includes(h[4]) && e.includes(h[5]);
}
export { throttle as a, parseSVG as i, highlString as n, require_notiflix_aio_3_2_8_min as o, include as r, Notify as t };
