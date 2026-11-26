export const CATEGORIES = [
  { value: "electric", label: "Электрика" },
  { value: "plumbing", label: "Сантехника" },
  { value: "earthworks", label: "Земляные работы" },
  { value: "repair", label: "Ремонт и отделка" },
  { value: "cargo", label: "Грузоперевозки" },
  { value: "other", label: "Другие работы" },
];

export const CATEGORY_LABELS = Object.fromEntries(
  CATEGORIES.map(({ value, label }) => [value, label])
);

export function renderCategoryOptions(includeAllOption = false, selectedValue = "") {
  const allOption = includeAllOption
    ? `<option value=""${selectedValue === "" ? " selected" : ""}>Все категории</option>`
    : "";
  return (
    allOption +
    CATEGORIES.map(
      ({ value, label }) =>
        `<option value="${value}"${value === selectedValue ? " selected" : ""}>${label}</option>`
    ).join("")
  );
}
