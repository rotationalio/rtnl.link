document.body.addEventListener('htmx:wsBeforeMessage', function(e) {
  data = JSON.parse(e.detail.message);
  const ctx = document.getElementById('links-chart');

  new Chart(ctx, {
    type: 'line',
    data: {
      labels: data.Time,
      datasets: [{
        label: '# of Visits',
        data: data.Views,
        borderWidth: 1
      }]
    },
    options: {
      responsive: true,
      scales: {
        y: {
          beginAtZero: true
        }
      }
    }
});
});