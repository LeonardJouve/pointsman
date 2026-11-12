export async function agent(element) {
  const response = await fetch("/agents");
  const body = await response.json();

  if (!response.ok) {
    element.textContent = body.error;

    return;
  }

  if (Object.keys(body).length === 0) {
    element.textContent = "No agents";

    return;
  }

  const list = document.createElement("div");
  list.className = "list-group";

  for (const [id, agent] of Object.entries(body)) {
    const item = document.createElement("details");
    item.className = "list-group-item";

    const name = document.createElement("summary");
    name.textContent = agent.name;

    const tools = document.createElement("ul");
    tools.append(...(agent.tools ?? []).map((tool) => {
      const li = document.createElement("li");
      li.textContent = tool;

      return li;
    }));

    item.append(name, attributes({id, ...agent, tools}));
    list.append(item);
  }

  element.append(list);
}

function attributes(values) {
  const list = document.createElement("dl");

  for (const label of ["id", "model", "prompt", "tools", "effort", "max_tokens"]) {
    if (values[label] === undefined) {
      continue;
    }

    const term = document.createElement("dt");
    term.textContent = label;

    const value = document.createElement("dd");
    value.append(values[label]);

    list.append(term, value);
  }

  return list;
}
