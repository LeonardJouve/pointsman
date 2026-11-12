import {request} from "./api.js";

export async function model(element) {
  const [body, error] = await request("/models");

  if (error) {
    element.textContent = error;

    return;
  }

  const list = document.createElement("ul");
  list.className = "list-group";

  for (const id of body) {
    const item = document.createElement("li");
    item.className = "list-group-item";
    item.textContent = id;
    list.append(item);
  }

  element.append(list);
}
