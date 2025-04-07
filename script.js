// script.js

// Пример данных JSON
const runnersData = [
  { id: 1, name: "Runner1", photo: "370195432.jpg", distance: 100 },
  { id: 2, name: "Runner2", photo: "370195432.jpg", distance: 75 },
  { id: 3, name: "Runner3", photo: "370195432.jpg", distance: 60 },
  { id: 4, name: "Runner4", photo: "370195432.jpg", distance: 50 },
  { id: 5, name: "Runner5", photo: "370195432.jpg", distance: 40 },
  { id: 6, name: "Runner6", photo: "370195432.jpg", distance: 30 },
];

// Формируем команды по 3 человека
const teams = [];
for (let i = 0; i < runnersData.length; i += 3) {
  teams.push(runnersData.slice(i, i + 3));
}

// Находим максимальное расстояние
const maxDistance = Math.max(...runnersData.map(runner => runner.distance));

// Заполняем таблицу
function fillRunnersTable() {
  const tableBody = document.getElementById("runners-table-body");
  runnersData.forEach(runner => {
    const percentage = ((runner.distance / maxDistance) * 100).toFixed(2);
    tableBody.innerHTML += `
      <tr>
        <td><img src="${runner.photo}" alt="${runner.name}"></td>
        <td>${runner.name}</td>
        <td>${runner.distance} км</td>
        <td>${percentage}%</td>
      </tr>
    `;
  });
}

// Заполняем блок команд
function fillTeamsContainer() {
  const container = document.getElementById("teams-container");
  teams.forEach((team, index) => {
    const totalDistance = team.reduce((sum, runner) => sum + runner.distance, 0);
    container.innerHTML += `
      <div class="col-md-4">
        <div class="team-card">
          <h4>Команда ${index + 1}</h4>
          <p>Общее расстояние: ${totalDistance} км</p>
          <ul>
            ${team.map(runner => `<li>${runner.name}: ${runner.distance} км</li>`).join("")}
          </ul>
        </div>
      </div>
    `;
  });
}

// Инициализация частиц
particlesJS.load('particles', 'particles.json', function() {
  console.log('Частицы загружены');
});

// Запуск функций
fillRunnersTable();
fillTeamsContainer();