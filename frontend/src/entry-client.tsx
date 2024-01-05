import { mount, StartClient } from "@solidjs/start/client"

import "./css/root.css";
import "./css/global.css";
import "./css/components.css";
import "./css/layout.css";

mount(() => <StartClient />, document.getElementById("app"))
