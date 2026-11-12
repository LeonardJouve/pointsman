import {field} from "./form.js";
import {request} from "./api.js";

const efforts = ["minimal", "low", "medium", "high", "xhigh", "max"];

export async function agentForm(element, onCreate) {
  const form = document.createElement("form");

  const name = document.createElement("input");
  name.className = "form-control";

  const model = document.createElement("select");
  model.className = "form-select";

  const prompt = document.createElement("textarea");
  prompt.className = "form-control";

  const tools = document.createElement("div");

  const effort = document.createElement("select");
  effort.className = "form-select";
  effort.append(...efforts.map((value) => new Option(value, value, value === "medium", value === "medium")));

  const maxTokens = document.createElement("input");
  maxTokens.className = "form-control";
  maxTokens.type = "number";
  maxTokens.min = 1;

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

  form.append(
    field("name", name),
    field("model", model),
    field("prompt", prompt),
    field("tools", tools),
    field("effort", effort),
    field("max_tokens", maxTokens),
    error,
    create,
    cancel,
  );
  element.append(form);

  form.addEventListener("submit", async (event) => {
    event.preventDefault();

    const agent = {
      name: name.value,
      model: model.value,
      prompt: prompt.value,
      tools: [...tools.querySelectorAll("input:checked")].map((input) => input.value),
      effort: effort.value,
    };

    if (maxTokens.value !== "") {
      agent.max_tokens = Number(maxTokens.value);
    }

    const [, failure] = await request("/agents", agent);
    if (failure) {
      error.textContent = failure;

      return;
    }

    error.textContent = "";
    form.reset();
    bootstrap.Modal.getOrCreateInstance(element.closest(".modal")).hide();
    onCreate();
  });

  const [models, modelsError] = await request("/models");
  if (modelsError) {
    model.replaceWith(modelsError);
  } else {
    model.append(...models.map((id) => new Option(id)));
  }

  const [schemas, toolsError] = await request("/tools");
  if (toolsError) {
    tools.textContent = toolsError;
  } else {
    tools.append(...schemas.map((schema) => checkbox(schema.Name)));
  }
}

function checkbox(value) {
  const box = document.createElement("input");
  box.className = "form-check-input";
  box.type = "checkbox";
  box.value = value;

  const label = document.createElement("label");
  label.className = "form-check-label";
  label.append(box, " ", value);

  const group = document.createElement("div");
  group.className = "form-check";
  group.append(label);

  return group;
}
