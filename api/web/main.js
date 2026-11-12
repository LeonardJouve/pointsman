import {agent} from "./agent.js";
import {agentForm} from "./agent-form.js";
import {model} from "./model.js";
import {session} from "./session.js";
import {sessionForm} from "./session-form.js";
import {tool} from "./tool.js";

const agents = document.querySelector("#agent-list");
const sessions = document.querySelector("#session-list");

agentForm(document.querySelector("#agent-form"), () => agent(agents));
agent(agents);
sessionForm(document.querySelector("#session-form"), () => session(sessions));
session(sessions);
model(document.querySelector("#model"));
tool(document.querySelector("#tool"));
