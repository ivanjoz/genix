// lib/unmarshall.ts
var unmarshall = (encoded) => {
  if (!Array.isArray(encoded) || encoded.length !== 2) {
    return encoded;
  }
  const [keysDef, content] = encoded;
  if (!Array.isArray(keysDef)) {
    return encoded;
  }
  const keysMap = {};
  for (const k of keysDef) {
    if (!Array.isArray(k)) continue;
    const typeId = k[0];
    const fields = {};
    let maxIndex = -1;
    for (let i = 1; i < k.length; i += 2) {
      const idx = k[i];
      const name = k[i + 1];
      fields[idx] = name;
      if (idx > maxIndex) maxIndex = idx;
    }
    keysMap[typeId] = { fields, maxIndex };
  }
  let lastTypeId = null;
  const decode = (val) => {
    if (!Array.isArray(val) || val.length === 0) {
      return val;
    }
    const header = val[0];
    if (header === 1) {
      if (val.length < 2) return val;
      const refBlock = val[1];
      if (!Array.isArray(refBlock)) return val;
      const typeId = refBlock[0];
      const skipIndices = /* @__PURE__ */ new Set();
      for (let i = 1; i < refBlock.length; i++) {
        skipIndices.add(refBlock[i]);
      }
      lastTypeId = typeId;
      return populate(typeId, val.slice(2), skipIndices);
    }
    if (header === 0) {
      if (lastTypeId === null) return val;
      let skipIndices = /* @__PURE__ */ new Set();
      let valueStartIdx = 1;
      if (Array.isArray(val[1])) {
        const sub = val[1];
        let isSkipBlock = false;
        if (sub.length > 0) {
          const h = sub[0];
          if (typeof h === "number" && h !== 0 && h !== 1 && h !== 2 && h !== 3) {
            isSkipBlock = true;
          } else {
            isSkipBlock = typeof h === "number" && h !== 0 && h !== 1 && h !== 2 && h !== 3;
          }
        }
        if (isSkipBlock) {
          for (const s of sub) {
            if (typeof s === "number") skipIndices.add(s);
          }
          valueStartIdx = 2;
        }
      }
      return populate(lastTypeId, val.slice(valueStartIdx), skipIndices);
    }
    if (header === 2) {
      const result = [];
      for (let i = 1; i < val.length; i++) {
        result.push(decode(val[i]));
      }
      return result;
    }
    if (header === 3) {
      const result = {};
      for (let i = 1; i < val.length; i += 2) {
        if (i + 1 < val.length) {
          const key = String(val[i]);
          result[key] = decode(val[i + 1]);
        }
      }
      return result;
    }
    return val.map(decode);
  };
  const populate = (typeId, values, skipIndices) => {
    const typeDef = keysMap[typeId];
    if (!typeDef) return values;
    const { fields, maxIndex } = typeDef;
    const obj = {};
    let valIdx = 0;
    for (let i = 0; i <= maxIndex; i++) {
      if (skipIndices.has(i)) {
        continue;
      }
      if (valIdx >= values.length) {
        break;
      }
      const fieldName = fields[i];
      if (fieldName) {
        obj[fieldName] = decode(values[valIdx]);
      }
      valIdx++;
    }
    return obj;
  };
  return decode(content);
};

// core/sharedHelpers.ts
var recreateObject = (obj, keysMap) => {
  if (Array.isArray(obj)) {
    return obj.map((x) => recreateObject(x, keysMap));
  }
  if (typeof obj !== "object" || !obj || !obj._) {
    return obj;
  }
  for (const [key, value] of Object.entries(obj)) {
    if (keysMap.has(key)) {
      const newKey = keysMap.get(key);
      if (newKey === key) {
        continue;
      }
      obj[newKey] = value;
      delete obj[key];
    }
  }
  for (let i = 0; i < obj._.length; i += 2) {
    const key = keysMap.get(obj._[i]);
    obj[key] = recreateObject(obj._[i + 1], keysMap);
  }
  delete obj._;
  return obj;
};

// workers/service-worker-cache.ts
var PRECACHE = "precache-v2";
var CACHE_ASSETS = "assets-v2";
var CACHE_STATIC = "static-v2";
var CACHE_APP = "app";
var HandlersMap = /* @__PURE__ */ new Map();
var PRECACHE_URLS = [
  // 'index.html',
  // './', // Alias for index.html
];
var parseFileExtension = (filename) => {
  const ix1 = filename.indexOf("?");
  if (ix1 !== -1) filename = filename.substring(0, ix1);
  const ix2 = filename.indexOf("@");
  if (ix2 !== -1) filename = filename.substring(0, ix2);
  filename = filename.substring(filename.indexOf("/", 8));
  const ix3 = filename.lastIndexOf(".");
  if (filename === "/" || (ix3 === -1 && filename[2]) === "/") {
    return [filename, "*"];
  }
  return [filename, filename.substring(ix3 + 1)];
};
self.addEventListener("install", (event) => {
  event.waitUntil(
    caches.open(PRECACHE).then((cache) => cache.addAll(PRECACHE_URLS)).then(() => self.skipWaiting())
  );
});
self.addEventListener("activate", (event) => {
  const currentCaches = [PRECACHE, CACHE_ASSETS, CACHE_STATIC, CACHE_APP];
  event.waitUntil(
    self.clients.claim()
    /*
    caches.keys().then(cacheNames => {
      return cacheNames.filter(cacheName => !currentCaches.includes(cacheName));
    }).then(cachesToDelete => {
      return Promise.all(cachesToDelete.map(cacheToDelete => {
        return caches.delete(cacheToDelete);
      }));
    }).then(() => self.clients.claim())
    */
  );
});
var extractVersion = (bodyHTML) => {
  const idx1 = bodyHTML.indexOf(`name="build-version"`);
  if (idx1 === -1) return null;
  const idx2 = bodyHTML.indexOf(`"`, idx1 + 28);
  const buildNumber = bodyHTML.substring(idx2 + 1, idx2 + 7);
  return buildNumber;
};
var versionToHour = (buildNumber) => {
  const nowTime = Math.floor(Date.now() / 1e3);
  const buildTime = parseInt(buildNumber.toLowerCase(), 36) * 10;
  const seconds = nowTime - buildTime;
  const hours = Math.floor(seconds / 60 / 60);
  const minutes = Math.floor((seconds - hours * 60 * 60) / 60);
  const horasText = `${hours} hora${hours === 1 ? "" : "s"}`;
  let hace = `${horasText} ${minutes} min`;
  if (hours === 0) hace = `${minutes} min`;
  if (hours > 12) hace = `m\xE1s de ${horasText}`;
  return hace;
};
var VersionInfo = { build: "", hasUpdated: false };
HandlersMap.set(99, async (message) => {
  return { ...message, response: "pong" };
});
self._isLocal = self.origin.includes("localhost") || self.origin.includes("127.0.0.1");
var lastNewVersionChecked = 0;
HandlersMap.set(7, async (message) => {
  const nowTime = Math.floor(Date.now() / 1e3);
  if (lastNewVersionChecked && nowTime - lastNewVersionChecked < 30) {
    console.log("Saltando revisi\xF3n de actualizaci\xF3n.");
    return {};
  }
  const versionCurrent = message.version;
  if (!versionCurrent) {
    console.log("No se envi\xF3 la version (build) a comparar.");
    return {};
  }
  const versionHasUpdated = await compareVersionUpdate(versionCurrent);
  lastNewVersionChecked = nowTime;
  return { versionHasUpdated };
});
var compareVersionUpdate = async (versionCurrent) => {
  VersionInfo.build = versionCurrent;
  const headers = new Headers();
  headers.append("pragma", "no-cache");
  headers.append("cache-control", "no-cache");
  try {
    const preResp = await fetch(self.location.origin + "/app-version", { method: "GET", headers });
    const bodyHTML = await preResp.text();
    const versionUpdated = extractVersion(bodyHTML) || "";
    VersionInfo.build = versionUpdated;
    console.log("build code compare fetch:: ", versionCurrent, " | ", versionUpdated);
    if (versionCurrent !== versionUpdated) {
      await caches.delete(CACHE_ASSETS);
      return versionToHour(versionUpdated);
    }
  } catch (error) {
    console.warn("Error al obtener la versi\xF3n nueva (HTML)::", error);
  }
  return "";
};
var clientIDsMap = /* @__PURE__ */ new Map();
var usedRequestIDs = /* @__PURE__ */ new Map();
var sendHandlers = /* @__PURE__ */ new Map();
var sendClientMessage = (clientID, content) => {
  const handler = sendHandlers.get(clientID);
  if (!handler) {
    console.log("No se encontr\xF3 el handler para el client:", clientID, "|", content);
  } else {
    handler(content);
  }
};
self.addEventListener("fetch", (event) => {
  const request = event.request;
  const url = new URL(event.request.url);
  if (url.pathname === "/_sw_") {
    event.respondWith((async () => {
      const accion = parseInt(url.searchParams.get("accion") || "0");
      const reqID = parseInt(url.searchParams.get("req") || "0");
      const enviroment = url.searchParams.get("env") || "main";
      if (!clientIDsMap.has(event.clientId)) {
        clientIDsMap.set(event.clientId, clientIDsMap.size + 1);
      }
      const clientID = clientIDsMap.get(event.clientId) || 0;
      const clientReqID = reqID * 1e3 + clientID;
      const usedReqTime = usedRequestIDs.get(clientReqID) || 0;
      if (usedReqTime && Date.now() - usedReqTime < 1e3) {
        const haceMs = Date.now() - usedReqTime;
        console.log("El id ", reqID, " est\xE1 duplicado. | Client:", event.clientId, "| Hace:", haceMs, "ms");
        return new Response(JSON.stringify({ "Error": "ReqID Duplicado." }), {
          headers: { "Content-Type": "application/json" }
        });
      }
      usedRequestIDs.set(reqID, Date.now());
      const client = await self.clients.get(event.clientId);
      if (!client) {
        console.warn(`No se encontr\xF3 el client con ID ${event.clientId}`);
        const msg = { error: `No se encontr\xF3 el client con ID ${event.clientId}` };
        return new Response(JSON.stringify(msg), {
          headers: { "Content-Type": "application/json" }
        });
      }
      sendHandlers.set(clientID, (content2) => {
        client.postMessage(content2);
      });
      const handler = HandlersMap.get(accion);
      if (!handler) {
        console.warn(`No se encontr\xF3 el handler para la acci\xF3n ${accion}`);
        const msg = { error: `No se encontr\xF3 el handler para la acci\xF3n ${accion}` };
        return new Response(JSON.stringify(msg), {
          headers: { "Content-Type": "application/json" }
        });
      }
      const requestClone = event.request.clone();
      const content = await requestClone.json();
      content.__enviroment__ = enviroment;
      content.__client__ = clientID;
      const response = await handler(content);
      const message = { ...response, __response__: accion, __req__: reqID };
      let info = "";
      if (accion === 3) {
        info = [content.route, content.cacheMode, reqID].join(" | ");
      }
      client.postMessage(message);
      console.log(`Respondiendo Fetch (${info}):`, parseObject(message));
      return new Response(JSON.stringify({ "ok": 1 }), {
        headers: { "Content-Type": "application/json" }
      });
    })());
    return;
  } else if (url.searchParams.has("__cache__")) {
    const cacheParam = url.searchParams.get("__cache__") || "";
    const [cacheTimeMinutes_, cacheVersion_, enviroment] = cacheParam.split(".");
    const cacheTimeMinutes = parseInt(cacheTimeMinutes_);
    const cacheVersion = parseInt(cacheVersion_);
    if (isNaN(cacheTimeMinutes) || isNaN(cacheVersion)) {
      console.warn(`Invalid __cache__ parameter format: ${cacheParam}. Bypassing cache.`);
      return fetch(event.request);
    }
    url.searchParams.delete("__cache__");
    const cacheKey = url.toString();
    console.log("buscando cach\xE9:", cacheKey);
    event.respondWith((async () => {
      const cache = await caches.open(`cache_req_${enviroment}`);
      const cachedResponse = await cache.match(cacheKey);
      const nowTime = Math.floor(Date.now() / 1e3);
      if (cachedResponse) {
        const cachedTimestamp = parseInt(cachedResponse.headers.get("x-cache-timestamp"));
        const cachedVersion = parseInt(cachedResponse.headers.get("x-cache-version"));
        if (cachedTimestamp && cachedVersion && nowTime - cachedTimestamp < cacheTimeMinutes * 60 && cachedVersion === cacheVersion) {
          console.log(`Serving from cache: ${cacheKey}`);
          return cachedResponse;
        } else {
          console.log(`Cache stale or version mismatch for ${cacheKey}. Fetching new.`);
        }
      }
      try {
        const networkResponse = await fetch(event.request);
        if (networkResponse.status === 200) {
          const responseToCache = networkResponse.clone();
          const headers = new Headers(responseToCache.headers);
          headers.set("x-cache-timestamp", String(nowTime));
          headers.set("x-cache-version", String(cacheVersion));
          const responseWithHeaders = new Response(await responseToCache.blob(), {
            status: responseToCache.status,
            statusText: responseToCache.statusText,
            headers
          });
          await cache.put(cacheKey, responseWithHeaders);
          console.log(`Fetched and cached: ${cacheKey}`);
        } else {
          console.log(`La consulta fall\xF3::`, networkResponse.status);
        }
        return networkResponse;
      } catch (error) {
        console.error(`Fetch failed for ${cacheKey}:`, error);
        if (cachedResponse) {
          console.log(`Network failed, serving stale cache for ${cacheKey}`);
          return cachedResponse;
        }
        throw error;
      }
    })());
  }
  if (self._isLocal) {
    return;
  }
  const contentType = request.headers.get("Content-Type");
  console.log("event URL:: ", request.url, "|", contentType);
  if (!request.url.startsWith(self.location.origin)) return;
  const [filename, ext] = parseFileExtension(request.url);
  if (filename === "/app-version") {
    return;
  }
  const requestURL = new URL(request.url);
  const isSameOrigin = requestURL.origin === self.location.origin;
  const isHTMLNavigation = request.mode === "navigate";
  const hasNoExtension = !requestURL.pathname.includes(".") || requestURL.pathname === "/";
  if (ext === "*") {
    console.log("Re-routing Service Worker:: ", filename);
  }
  const CACHE_NAME = ["js", "css", "html", "*", "ts", "mjs", "tsx"].includes(ext) ? CACHE_ASSETS : CACHE_STATIC;
  const OFFLINE_URL = "/";
  if (isSameOrigin && isHTMLNavigation && hasNoExtension) {
    event.respondWith(
      caches.open(CACHE_NAME).then((cache) => {
        return cache.match(OFFLINE_URL).then((cachedResponse) => {
          if (cachedResponse && !navigator.onLine) {
            console.log(`[Service Worker] Serving cached HTML for ${request.url} from ${OFFLINE_URL}`);
            return cachedResponse;
          }
          console.log(`[Service Worker] HTML not in cache, fetching from network: ${request.url}`);
          return fetch(request).then((networkResponse) => {
            const contentType2 = networkResponse.headers.get("Content-Type");
            if (networkResponse.ok && contentType2 && contentType2.includes("text/html")) {
              console.log(`[Service Worker] Caching new HTML response from ${request.url} as ${OFFLINE_URL}`);
              cache.put(OFFLINE_URL, networkResponse.clone());
            }
            return networkResponse;
          }).catch((error) => {
            if (cachedResponse) {
              return cachedResponse;
            }
            console.error(`[Service Worker] Network failed for HTML navigation: ${request.url}`, error);
            return new Response("<h1>Offline</h1><p>You appear to be offline and this content is not cached.</p>", {
              headers: { "Content-Type": "text/html" }
            });
          });
        });
      })
    );
    return;
  }
  event.respondWith(
    caches.open("cache_").then((cache) => {
      return cache.match(request).then((cachedResponse) => {
        if (cachedResponse) {
          const contentType2 = cachedResponse.headers.get("content-type") || "";
          if (!contentType2.includes("/html")) {
            console.log("[Service Worker] Serving from cache:", request.url);
            return cachedResponse;
          } else {
            console.log("Es HTML!!", ext);
          }
        }
        return fetch(request).then((response) => {
          if (!response || response.status !== 200 || response.type !== "basic") {
            return response;
          }
          const contentType2 = request.headers.get("Content-Type");
          console.log("Guardando en cach\xE9: ", CACHE_NAME, " | " + request.url, " | ", contentType2);
          const responseToCache = response.clone();
          console.log("[Service Worker] Caching new asset:", request.url);
          cache.put(request, responseToCache);
          return response;
        }).catch((error) => {
          console.error("[Service Worker] Fetch failed:", request.url, error);
          return new Response(
            "Network error occurred.",
            { status: 503, statusText: "Service Unavailable" }
          );
        });
      });
    })
  );
});
var appMemoryCache = /* @__PURE__ */ new Map();
var hasCacheKey = async (key, cache) => {
  const cacheName = cache ? cache + "_" + CACHE_APP : CACHE_APP;
  const appCache = await caches.open(cacheName);
  const hasCache = await appCache.match(key);
  const memoryCache = appMemoryCache.get(cacheName);
  if (!hasCache && memoryCache) {
    memoryCache.delete(key);
  }
  return hasCache;
};
var getCacheRecord = async (key, cache) => {
  const cacheName = cache ? cache + "_" + CACHE_APP : CACHE_APP;
  if (!appMemoryCache.has(cacheName)) {
    appMemoryCache.set(cacheName, /* @__PURE__ */ new Map());
  }
  const memoryCache = appMemoryCache.get(cacheName);
  if (memoryCache?.has(key)) {
    const cacheExists = await caches.has(cacheName);
    if (cacheExists) {
      return memoryCache.get(key);
    } else {
      appMemoryCache.set(cacheName, /* @__PURE__ */ new Map());
    }
  }
  const appCache = await caches.open(cacheName);
  const startTime = Date.now();
  const response = await appCache.match(key);
  if (response) {
    let jsonResponse = await response.json();
    if (jsonResponse.__keys__) {
      const keysMap = new Map(jsonResponse.__keys__);
      const keysMapReversed = new Map([...keysMap.entries()].map((x) => [x[1], x[0]]));
      jsonResponse = recreateObject(jsonResponse.content, keysMapReversed);
    }
    console.log(`Cache response "${key}" in ${Date.now() - startTime}ms (v2)`);
    return jsonResponse;
  }
  return void 0;
};
var setCacheRecord = async (key, content, cache) => {
  const cacheName = cache ? cache + "_" + CACHE_APP : CACHE_APP;
  if (!appMemoryCache.has(cacheName)) {
    appMemoryCache.set(cacheName, /* @__PURE__ */ new Map());
  }
  const appCache = await caches.open(cacheName);
  const memoryCache = appMemoryCache.get(cacheName);
  if (!content) {
    memoryCache?.delete(key);
    await appCache.delete(key);
    return;
  }
  memoryCache?.set(key, content);
  const startTime = Date.now();
  if (typeof content !== "string") {
    content = JSON.stringify(content);
  }
  const response = new Response(content, {
    headers: {
      "Content-Type": "application/json",
      "Content-Length": String(content.length)
    }
  });
  await appCache.put(key, response);
  console.log(`Cache put "${key}" in ${Date.now() - startTime}ms (v2)`);
};
var parseObject = (rec) => {
  const newObject = {};
  for (const key in rec) {
    const values = rec[key];
    if (typeof values === "number" || typeof values === "string") {
      newObject[key] = values;
    } else if (Array.isArray(values)) {
      newObject[key] = `[${values.length}]`;
    } else if (values && typeof values === "object") {
      newObject[key] = `{${Object.keys(values).join(", ")}}`;
    }
  }
  return newObject;
};

// workers/service-worker.ts
var parseResponseAsStream = async (fetchResponse, props) => {
  const contentType = fetchResponse.headers.get("Content-Type");
  if (fetchResponse.status && props.status) {
    props.status.code = fetchResponse.status;
    props.status.message = fetchResponse.statusText;
  }
  if (fetchResponse.status === 200) {
    const reader = fetchResponse.body?.getReader();
    const stream = new ReadableStream({
      start(controller) {
        return pump();
        function pump() {
          return reader.read().then(({ done, value }) => {
            if (done) {
              controller.close();
              return Promise.resolve();
            }
            sendClientMessage(props.__client__, { __response__: 5, bytes: value.length });
            controller.enqueue(value);
            return pump();
          });
        }
      }
    });
    const responseStream = new Response(stream);
    const responseText = await responseStream.text();
    if (props) {
      props.contentLength = responseText.length;
    }
    return Promise.resolve(JSON.parse(responseText));
  } else if (fetchResponse.status === 401) {
    console.warn("Error 401, la sesi\xF3n ha expirado.");
  } else if (fetchResponse.status !== 200) {
    console.log(fetchResponse);
    if (!contentType || contentType.indexOf("/json") === -1) {
      console.log("Parseando como texto");
      return fetchResponse.text();
    } else {
      console.log("parseando como JSON");
      return fetchResponse.json();
    }
  }
};
var extractUpdated = (obj, useMin) => {
  const updatedStatus = {};
  for (const [key, values] of Object.entries(obj)) {
    if (values && !Array.isArray(values)) {
      continue;
    }
    let maxOrMin = 0;
    for (const v of values || []) {
      const updated = v.updated || v.upd || 0;
      if (useMin) {
        if (maxOrMin === 0 || updated < maxOrMin) {
          maxOrMin = updated;
        }
      } else {
        if (updated > maxOrMin) {
          maxOrMin = updated;
        }
      }
    }
    updatedStatus[key] = maxOrMin;
  }
  return updatedStatus;
};
var addToRoute = (route, key, value) => {
  const sign = route.includes("?") ? "&" : "?";
  if (typeof value === "string") {
    value = value.replace("?", "&");
  }
  return route + `${sign}${key}=${value}`;
};
var forceFetch = false;
var forcedFetchRequests = /* @__PURE__ */ new Set();
HandlersMap.set(11, async () => {
  forceFetch = true;
  forcedFetchRequests = /* @__PURE__ */ new Set();
  setTimeout(() => {
    forceFetch = false;
  }, 8e3);
  return { ok: 1 };
});
HandlersMap.set(3, async (args) => {
  return await fetchCache(args);
});
var makeKey = (args) => {
  const key = [args.route, args.partition?.value || "0"].join("_");
  const cacheName = [args.__enviroment__ || "main", args.module || "?"].join("_");
  return [key, cacheName];
};
HandlersMap.set(12, async (args) => {
  const [key, cacheName] = makeKey(args);
  const lastSync = await getCacheRecord(key + "_updated", cacheName);
  console.log("lastSync obtenido (1):", args, args.route, key, lastSync);
  const updatedStatus = lastSync?.updatedStatus || {};
  return { updatedStatus, updated: lastSync?.fetchTime || 0 };
});
var getRecordKeys = (args, field) => {
  const keyIDs = args.keysIDs && Object.hasOwn(args.keysIDs, field) ? args.keysIDs[field] : args.keyID || "id";
  return keyIDs;
};
var getKeyValue = (record, keyIDs) => {
  if (typeof keyIDs === "string") {
    return record[keyIDs] || record.ID || 0;
  } else if (Array.isArray(keyIDs)) {
    const arr = [];
    for (const key of keyIDs) {
      if (record[key]) {
        arr.push(record[key]);
      } else {
        return 0;
      }
    }
    return arr.join("_");
  } else {
    return 0;
  }
};
var handleFetchResponse = async (args, lastSync, response, nowTime) => {
  if (Array.isArray(response)) {
    response = { _default: response };
  }
  const updatedStatusDelta = extractUpdated(response);
  const updatedMinDelta = extractUpdated(response, true);
  let hasChanged = false;
  for (const [key2, updated] of Object.entries(updatedStatusDelta)) {
    if (!updated) {
      continue;
    }
    const prevUpdated = lastSync.updatedStatus[key2];
    if (updated !== prevUpdated) {
      lastSync.updatedStatus[key2] = updated;
      hasChanged = true;
    }
    if (updatedMinDelta[key2] < prevUpdated) {
      console.warn(`Cache Error: En "${args.route}" [${key2}] se est\xE1n obteniendo registros con [updated] menor que el cach\xE9 (${(response[key2] || []).length} recibidos)`);
    }
  }
  console.log("Fetch cache ha cambiado?:", hasChanged, "|", args.route, "|", updatedStatusDelta);
  const [key, cacheName] = makeKey(args);
  const keyUpdated = key + "_updated";
  if (!hasChanged && args.cacheMode !== "updateOnly") {
    response = await getCacheRecord(key, cacheName);
  } else if (hasChanged) {
    const prevResponse = await getCacheRecord(key, cacheName);
    console.log("cache obtenido:", key, cacheName, args);
    if (self._isLocal) {
      console.log("prevResponse (old)", args.route, args.cacheMode, { ...prevResponse });
    }
    const usedKeysCount = {};
    for (const [respKey, newRecords] of Object.entries(response)) {
      if ((newRecords || []).length === 0) {
        continue;
      }
      const keyIDs = getRecordKeys(args, respKey);
      let missingCount = 0;
      usedKeysCount[String(keyIDs)] = (usedKeysCount[String(keyIDs)] || 0) + newRecords.length;
      const makeKeyID = (r) => {
        const value = getKeyValue(r, keyIDs);
        if (!value) {
          missingCount++;
        }
        return value;
      };
      const prevRecords = prevResponse[respKey] || [];
      if (!Array.isArray(prevRecords)) {
        continue;
      }
      const recordsMap = /* @__PURE__ */ new Map();
      for (const e of prevRecords) {
        recordsMap.set(makeKeyID(e), e);
      }
      for (const e of newRecords) {
        recordsMap.set(makeKeyID(e), e);
      }
      if (missingCount > 0) {
        console.warn(`Cache Error: En "${args.route}" (${respKey}) hay ${missingCount} registros sin la key: ${keyIDs}`);
      }
      console.log("Cache usedKeysCount", usedKeysCount);
      const mergedRecords = [];
      for (const e of recordsMap.values()) {
        mergedRecords.push(e);
      }
      prevResponse[respKey] = mergedRecords;
    }
    if (self._isLocal) {
      console.log("prevResponse (new)", args.route, args.cacheMode, { ...prevResponse });
    }
    setCacheRecord(key, prevResponse, cacheName);
    response = prevResponse;
  } else {
    response = null;
  }
  lastSync.fetchTime = nowTime;
  setCacheRecord(keyUpdated, lastSync, cacheName);
  return response;
};
var acknowledgeResponses = /* @__PURE__ */ new Set();
HandlersMap.set(21, async (args) => {
  const reqID = (args.__req__ || 0) * 1e3 + args.__client__;
  acknowledgeResponses.add(reqID);
});
var fetchCache = async (args) => {
  console.log("Obteniendo fetch service worker:", args.route, "|", args.cacheMode, "|", args.__req__, "|", args.__version__);
  const [key, cacheName] = makeKey(args);
  const keyUpdated = key + "_updated";
  const makeStats = (content) => {
    if (!content || Object.keys(content).length === 0) {
      return ["sin registros"];
    }
    const stats = [];
    for (const key2 of Object.keys(content)) {
      stats.push(`${key2}=${(content[key2] || []).length}`);
    }
    return stats;
  };
  const lastSyncEmpty = {
    fetchTime: 0,
    updatedStatus: {},
    __version__: args.__version__
  };
  let lastSync = await getCacheRecord(keyUpdated, cacheName) || lastSyncEmpty;
  if (lastSync.fetchTime && lastSync.__version__ !== args.__version__) {
    console.log(`Linmpiando cach\xE9, difererente versi\xF3n ${args.__version__} > ${lastSync.__version__}. Route ${args.route}`);
    lastSync = lastSyncEmpty;
    await setCacheRecord(keyUpdated, lastSync, cacheName);
    await setCacheRecord(key, null, cacheName);
  } else {
    const isCachePresent = await hasCacheKey(key, cacheName);
    if (lastSync.fetchTime && !isCachePresent) {
      await setCacheRecord(keyUpdated, lastSyncEmpty, cacheName);
      lastSync = lastSyncEmpty;
    } else if (isCachePresent && !lastSync.fetchTime) {
      await setCacheRecord(key, null, cacheName);
    }
  }
  if (args.cacheMode === "offline") {
    const content = await getCacheRecord(key, cacheName);
    if (content && content.__version__ === args.__version__) {
      const updatedStatus = extractUpdated(content);
      for (const [key2, updated] of Object.entries(updatedStatus)) {
        let reSave = false;
        if (lastSync.updatedStatus[key2] !== updated) {
          console.log("El updatedStatus difiere:", args.route, "|", key2, "|", lastSync.updatedStatus[key2], " vs ", updated);
          lastSync.updatedStatus[key2] = updated;
          reSave = true;
        }
        if (reSave) {
          setCacheRecord(keyUpdated, lastSync, cacheName);
        }
      }
    }
    console.log("Enviando fetch response (offline):", args.route);
    console.log(`${args.route}: Retornando registros "${args.cacheMode || "normal"}". ${makeStats(content).join(" | ")}`);
    return { content: content?._default ? content._default : content };
  }
  const fetchTime = Math.floor(Date.now() / 1e3);
  args.status = { code: 200, message: "" };
  try {
    let route = args.routeParsed || args.route;
    if (args.partition && args.partition.value) {
      const param = args.partition.param || args.partition.key;
      route = addToRoute(route, param, args.partition.value);
    }
    const hasCache = lastSync && lastSync.updatedStatus && lastSync.fetchTime;
    console.log("hasCache", args.route, lastSync);
    const fields = args.fields || [];
    for (const field of fields) {
      const updated = lastSync.updatedStatus[field] || 0;
      if (!updated) {
        route = addToRoute(route, field, 0);
      }
    }
    if (hasCache) {
      if (lastSync.updatedStatus._default) {
        route = addToRoute(route, "updated", lastSync.updatedStatus._default);
      } else {
        let minUpdated = 0;
        for (const [key2, updated] of Object.entries(lastSync.updatedStatus)) {
          if (minUpdated === 0 || updated < minUpdated) {
            minUpdated = updated;
          }
          if (fields.length > 0 && !fields.includes(key2)) {
            continue;
          }
          route = addToRoute(route, key2, updated);
        }
        route = addToRoute(route, "updated", minUpdated);
      }
      const cacheSyncTime = args.cacheSyncTime || args.useCache?.min || 0;
      const fetchNextTime = lastSync.fetchTime + cacheSyncTime * 60;
      const remainig = fetchNextTime - fetchTime;
      let doFetch = args.cacheMode === "refresh";
      if (lastSync.forceNetwork) {
        console.log("Forzando fetch por flag forceNetwork en ILastSync:", args.route);
        doFetch = true;
        lastSync.forceNetwork = false;
        await setCacheRecord(keyUpdated, lastSync, cacheName);
      } else if (forceFetch && !forcedFetchRequests.has(key)) {
        console.log("Forzando fetch por flag global:", args.route);
        doFetch = true;
      } else if (remainig <= 0) {
        console.log("Preparando fetch: ", args.route, " | Last: ", lastSync.fetchTime, "| Remainig:", remainig);
        doFetch = true;
      }
      for (const field of args.fields || []) {
        const updated = lastSync.updatedStatus[field] || 0;
        if (!updated) {
          doFetch = true;
        }
      }
      if (!doFetch) {
        const remainig2 = fetchNextTime - fetchTime;
        console.log(`Obviando sync fech "${key}". Quedan ${remainig2}s`);
        if (args.cacheMode === "updateOnly") {
          console.log(args.route, "Retornando null por 'updated only'");
          return { content: null };
        } else {
          const content = await getCacheRecord(key, cacheName);
          console.log(`${args.route}: Retornando registros "${args.cacheMode || "normal"}`, parseObject(content));
          return {
            content: content?._default ? content._default : content
          };
        }
      }
    }
    console.log(`Realizando fetch (${route})...`);
    const preResponse = await self.fetch(route, { headers: args.headers });
    if (preResponse.status && preResponse.status !== 200) {
      const responseText = await preResponse.text();
      return { error: responseText };
    }
    let response = await parseResponseAsStream(preResponse, args) || {};
    console.log("response pre-unmarshall", response);
    response = unmarshall(response);
    console.log("response post-unmarshall", response);
    if (Array.isArray(response.response) && typeof response.message === "string") {
      response = response.response;
    }
    if (Array.isArray(response)) {
      response = { _default: response };
    }
    response.__version__ = args.__version__;
    console.log(`Fetch response recibida! (${route}) | Has-cach\xE9: ${hasCache}`);
    if (hasCache) {
      console.log("prevResponse (hasCache)", args.route, args.cacheMode, { ...response });
      response = await handleFetchResponse(args, lastSync, response, fetchTime);
    } else {
      const updatedStatus = extractUpdated(response);
      Object.assign(lastSync, { fetchTime, updatedStatus });
      setCacheRecord(keyUpdated, lastSync, cacheName);
      console.log("prevResponse (recent)", args.route, args.cacheMode, { ...response });
      setCacheRecord(key, response, cacheName);
    }
    console.log(`${args.route}: Retornando registros "${args.cacheMode || "normal"}". ${makeStats(response).join(" | ")}`);
    return { content: response };
  } catch (error) {
    console.log("Fetch Error::", error);
    return { error };
  }
};
HandlersMap.set(13, async (args) => {
  const [keyUpdated, cacheName] = makeKey(args.args) + "_updated";
  const lastFech = await getCacheRecord(keyUpdated, cacheName) || {
    fetchTime: 0,
    updatedStatus: {}
  };
  const nowTime = Math.floor(Date.now() / 1e3) - 5;
  args.args.__enviroment__ = args.__enviroment__;
  console.log("Guardando external fech response en cach\xE9:", args.args.route);
  handleFetchResponse(args.args, lastFech, args.response, nowTime);
});
HandlersMap.set(14, async (args) => {
  const [key, cacheName] = makeKey(args);
  const keyUpdated = key + "_updated";
  const lastSync = await getCacheRecord(keyUpdated, cacheName) || { fetchTime: 0, updatedStatus: {} };
  lastSync.forceNetwork = true;
  await setCacheRecord(keyUpdated, lastSync, cacheName);
  console.log(`ForceNetwork set to true for key: ${keyUpdated}`);
  return { ok: 1 };
});
HandlersMap.set(15, async (args) => {
  const keyArgs = {
    route: args.route,
    module: args.module,
    partition: { value: args.partValue }
  };
  const [key, cacheName] = makeKey(keyArgs);
  const response = await getCacheRecord(key, cacheName);
  if (!response) {
    return [];
  }
  const records = response[args.propInResponse || "_default"];
  if (!records) {
    return [];
  }
  if (args.filter) {
    const filterKeyValues = args.filter.split(",").filter((x) => x).map((x) => {
      const filter = x.split("=");
      if (!isNaN(filter[1])) {
        filter.push(parseInt(filter[1]));
      } else {
        filter.push(0);
      }
      return filter;
    });
    console.log("keyValues filters:", filterKeyValues);
    const filtered = [];
    for (const r of records) {
      for (const fil of filterKeyValues) {
        const value = r[fil[0]];
        if (value === fil[1] || value === fil[2]) {
          filtered.push(r);
        }
      }
    }
    return filtered;
  } else {
    return response;
  }
});
HandlersMap.set(22, async (args) => {
  console.log("obteniendo cantidad de registros obtenidos");
  const cacheStores = await caches.keys();
  console.log("cache stores::", cacheStores, "| Env:", args.__enviroment__);
  const cacheStats = [];
  for (const name of cacheStores) {
    if (!name.includes("_")) {
      continue;
    }
    const [envirotment, module] = name.split("_");
    if (envirotment !== args.__enviroment__) {
      continue;
    }
    cacheStats.push({ name, module, size: 0 });
  }
  await Promise.all(cacheStats.map((e) => {
    return new Promise((resolve, reject) => {
      let cache;
      caches.open(e.name).then((_cache) => {
        cache = _cache;
        return cache.keys();
      }).then((requests) => {
        return Promise.all(requests.map(async (request) => {
          try {
            const response = await cache.match(request);
            if (response) {
              const contentLength = response.headers.get("Content-Length") || response.headers.get("content-length");
              if (!e.size) {
                e.size = 0;
              }
              e.size += parseInt(contentLength || "0", 10);
            } else {
              console.warn(`Service Worker: No matching response found for request in cache '${e.name}':`, request.url);
            }
          } catch (itemError) {
            console.error(`Service Worker: Error matching request or getting size for ${request.url}:`, itemError);
          }
        }));
      }).then(() => {
        resolve(0);
      }).catch((err) => {
        console.log("Error al obtener informacion del cach\xE9:", err);
        reject(err);
      });
    });
  }));
  console.log("cacheStats", cacheStats);
  return { cacheStats };
});
HandlersMap.set(23, async (args) => {
  console.log(`Eliminando cach\xE9 "${args.cacheName}" (Enviroment ${args.__enviroment__})...`);
  await caches.delete(args.cacheName);
  console.log(`Cach\xE9 "${args.cacheName}" eliminado! (Enviroment ${args.__enviroment__})...`);
  return { ok: 1 };
});
HandlersMap.set(26, async (args) => {
  console.log("Eliminando cach\xE9...");
  const cacheNames = await caches.keys();
  for (const name of cacheNames) {
    if (name.startsWith(args.__enviroment__)) {
      console.log(`Eliminando cach\xE9 por ambiente "${args.__enviroment__}": ${name}`);
      await caches.delete(name);
    }
  }
  console.log("Cach\xE9 eliminado.");
  return { ok: 1 };
});
HandlersMap.set(24, async (args) => {
  const cacheName = [args.__enviroment__ || "main", args.module || "?"].join("_");
  console.log("Setting ForceNerwork for routes: ", args.routes);
  const cache = await caches.open(cacheName + "_" + CACHE_APP);
  const requests = await cache.keys();
  const routesUpdated = /* @__PURE__ */ new Set();
  for (const request of requests) {
    const cachedRoute = request.url.split("/")[3];
    if (!cachedRoute) {
      continue;
    }
    console.log("buscando request::", cachedRoute);
    for (const route of args.routes) {
      if (cachedRoute.startsWith(route) && cachedRoute.endsWith("_updated")) {
        const lastSync = await getCacheRecord(cachedRoute, cacheName);
        lastSync.forceNetwork = true;
        await setCacheRecord(cachedRoute, lastSync, cacheName);
        routesUpdated.add(route);
        break;
      }
    }
  }
  console.log(`ForceNetwork = true for routes: ${[...routesUpdated].join(", ")} | In ${requests.length}`);
  return { ok: 1 };
});
//# sourceMappingURL=sw.js.map
