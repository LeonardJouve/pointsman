import {request} from "./api.js";

const colors = {
  running: "bg-warning",
  succeeded: "bg-success",
  failed: "bg-danger",
};

export async function session(element) {
  const [body, error] = await request("/sessions");

  if (error) {
    element.textContent = error;

    return;
  }

  if (Object.keys(body).length === 0) {
    element.textContent = "No sessions";

    return;
  }

  const list = document.createElement("div");
  list.className = "list-group";

  const sessions = Object.values(body).sort((a, b) => new Date(b.created) - new Date(a.created));

  for (const found of sessions) {
    const item = document.createElement("details");
    item.className = "list-group-item";

    const status = document.createElement("span");
    status.className = `rounded-circle p-2 position-absolute top-50 end-0 translate-middle-y ${colors[found.status]}`;
    status.title = found.status;

    const word = document.createElement("span");
    word.className = "visually-hidden";
    word.textContent = found.status;
    status.append(word);

    const title = document.createElement("summary");
    title.className = "position-relative pe-4";
    title.append(status, `${found.agent.name} ${new Date(found.created).toLocaleString()}`);

    item.append(title);
    list.append(item);
  }

  element.replaceChildren(list);
}
