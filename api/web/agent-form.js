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

    const response = await fetch("/agents", {method: "POST", body: JSON.stringify(agent)});
    if (!response.ok) {
      error.textContent = (await response.json()).error;

      return;
    }

    error.textContent = "";
    form.reset();
    bootstrap.Modal.getOrCreateInstance(element.closest(".modal")).hide();
    onCreate();
  });

  const models = await fetch("/models");
  const modelsBody = await models.json();
  if (models.ok) {
    model.append(...modelsBody.map((id) => new Option(id)));
  } else {
    model.replaceWith(modelsBody.error);
  }

  const toolsResponse = await fetch("/tools");
  const toolsBody = await toolsResponse.json();
  if (toolsResponse.ok) {
    tools.append(...toolsBody.map((schema) => checkbox(schema.Name)));
  } else {
    tools.textContent = toolsBody.error;
  }
}

function field(label, input) {
  const group = document.createElement("div");
  group.className = "mb-3";

  const text = document.createElement("label");
  text.className = "form-label";
  text.textContent = label;

  group.append(text, input);

  return group;
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
