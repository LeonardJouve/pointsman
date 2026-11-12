import {request} from "./api.js";

export async function tool(element) {
  const [body, error] = await request("/tools");

  if (error) {
    element.textContent = error;

    return;
  }

  const list = document.createElement("div");
  list.className = "list-group";

  for (const schema of body) {
    const item = document.createElement("details");
    item.className = "list-group-item";

    const name = document.createElement("summary");
    name.textContent = schema.Name;

    const description = document.createElement("p");
    description.textContent = schema.Description;

    const parameters = document.createElement("pre");
    parameters.textContent = JSON.stringify(schema.Parameters, null, 2);

    item.append(name, description, parameters);
    list.append(item);
  }

  element.append(list);
}
