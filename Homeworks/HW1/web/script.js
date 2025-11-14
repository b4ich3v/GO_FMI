const DELIM = ';';

function parseCsv(text) {
  const lines = text.trim().split(/\r?\n/);
  return lines.map(line => line.split(DELIM));
}

function buildTable(data) {
  const table = document.getElementById('stats-table');
  const thead = table.querySelector('thead');
  const tbody = table.querySelector('tbody');

  thead.innerHTML = '';
  tbody.innerHTML = '';

  if (!data.length) return;

  const [header, ...rows] = data;

  // Header
  const trHead = document.createElement('tr');
  header.forEach(col => {
    const th = document.createElement('th');
    th.textContent = col;
    trHead.appendChild(th);
  });
  thead.appendChild(trHead);

  // Body
  rows.forEach(row => {
    const tr = document.createElement('tr');
    row.forEach((cell, idx) => {
      const td = document.createElement('td');
      const colName = header[idx];

      if (colName === 'Repos' || colName === 'Followers' || colName === 'Forks') {
        td.className = 'num';
      } else if (colName === 'Languages') {
        td.className = 'languages';
      } else if (colName === 'Activity') {
        td.className = 'activity';
      } else if (colName === 'User') {
        td.className = 'user';
      }

      td.textContent = cell;
      tr.appendChild(td);
    });
    tbody.appendChild(tr);
  });
}

async function loadCsv() {
  const status = document.getElementById('status');
  try {
    const res = await fetch('out.csv');
    if (!res.ok) {
      status.textContent =
        'Failed to load out.csv (HTTP ' + res.status + '). Make sure the file exists.';
      return;
    }
    const text = await res.text();
    const data = parseCsv(text);
    buildTable(data);
    status.textContent = 'The data is loaded successfully from out.csv.';
  } catch (err) {
    console.error(err);
    status.textContent =
      'An error occurred while reading out.csv. See the console (F12) for details.';
  }
}

loadCsv();
