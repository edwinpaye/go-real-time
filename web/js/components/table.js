// Reusable Reactive Table Component

export class DataTable {
  static create({ columns, rows, rowIdKey = 'id', onRenderRow }) {
    const tableContainer = document.createElement('div');
    tableContainer.className = 'table-container';

    const table = document.createElement('table');
    table.className = 'data-table';

    // Thead
    const thead = document.createElement('thead');
    const headerRow = document.createElement('tr');
    columns.forEach(col => {
      const th = document.createElement('th');
      th.textContent = col.label;
      if (col.width) th.style.width = col.width;
      headerRow.appendChild(th);
    });
    thead.appendChild(headerRow);
    table.appendChild(thead);

    // Tbody
    const tbody = document.createElement('tbody');
    rows.forEach(item => {
      const tr = document.createElement('tr');
      tr.dataset.productId = item[rowIdKey];

      if (item.is_deleted) {
        tr.className = 'row-deleted-danger';
      }

      if (onRenderRow) {
        tr.innerHTML = onRenderRow(item);
      }
      tbody.appendChild(tr);
    });
    table.appendChild(tbody);
    tableContainer.appendChild(table);

    return tableContainer;
  }
}
