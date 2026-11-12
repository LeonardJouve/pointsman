export async function model(element) {
  const response = await fetch("/models");
  const body = await response.json();

  if (!response.ok) {
    element.textContent = body.error;

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
