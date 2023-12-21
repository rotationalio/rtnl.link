document.body.addEventListener('htmx:wsBeforeMessage', function(e) {
  data = JSON.parse(e.detail.message);
  const ctx = document.getElementById('link-detail-chart');
  
  new Chart(ctx, {
    type: 'line',
    data: {
      labels: data.Time,
      datasets: [{
        label: 'URL Activity',
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
