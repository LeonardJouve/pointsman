export function field(label, input) {
  const group = document.createElement("div");
  group.className = "mb-3";

  const text = document.createElement("label");
  text.className = "form-label";
  text.textContent = label;

  group.append(text, input);

  return group;
}
