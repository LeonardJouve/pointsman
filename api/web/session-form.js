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
    const response = await fetch("/agents");
    const body = await response.json();

    if (!response.ok) {
      agent.replaceChildren();
      error.textContent = body.error;

      return;
    }

    error.textContent = "";
    agent.replaceChildren(...Object.entries(body).map(([id, found]) => new Option(found.name || id, id)));
  });

  form.addEventListener("submit", async (event) => {
    event.preventDefault();

    const body = {agent: agent.value, prompt: prompt.value};
    const response = await fetch("/sessions", {method: "POST", body: JSON.stringify(body)});
    if (!response.ok) {
      error.textContent = (await response.json()).error;

      return;
    }

    error.textContent = "";
    form.reset();
    bootstrap.Modal.getOrCreateInstance(modal).hide();
    onCreate();
  });
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
