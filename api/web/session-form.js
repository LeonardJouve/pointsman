import {field} from "./form.js";
import {request} from "./api.js";

export function sessionForm(element, onCreate) {
  const form = document.createElement("form");

  const agent = document.createElement("select");
  agent.className = "form-select";

  const prompt = document.createElement("textarea");
  prompt.className = "form-control";

  const error = document.createElement("p");
  error.className = "text-danger";

  const create = document.createElement("button");
  create.className = "btn btn-primary";
  create.textContent = "Create";

  const cancel = document.createElement("button");
  cancel.className = "btn btn-secondary ms-2";
  cancel.type = "button";
  cancel.textContent = "Cancel";
  cancel.dataset.bsDismiss = "modal";

  form.append(field("agent", agent), field("prompt", prompt), error, create, cancel);
  element.append(form);

  const modal = element.closest(".modal");

  modal.addEventListener("show.bs.modal", async () => {
    const [agents, failure] = await request("/agents");

    if (failure) {
      agent.replaceChildren();
      error.textContent = failure;

      return;
    }

    error.textContent = "";
    agent.replaceChildren(...Object.entries(agents).map(([id, found]) => new Option(found.name || id, id)));
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();

    const [, failure] = await request("/sessions", {agent: agent.value, prompt: prompt.value});
    if (failure) {
      error.textContent = failure;

      return;
    }

    error.textContent = "";
    form.reset();
    bootstrap.Modal.getOrCreateInstance(modal).hide();
    onCreate();
  });
}
