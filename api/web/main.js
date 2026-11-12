import {agent} from "./agent.js";
import {agentForm} from "./agent-form.js";
import {model} from "./model.js";
import {tool} from "./tool.js";

const agents = document.querySelector("#agent-list");

agentForm(document.querySelector("#agent-form"), () => agent(agents));
agent(agents);
model(document.querySelector("#model"));
tool(document.querySelector("#tool"));
