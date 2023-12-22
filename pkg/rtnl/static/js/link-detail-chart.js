function createWebSocket(url) {
  var sock = new WebSocket(url);
  sock.binaryType = "blob";
  sock.onerror = function(e) {
    console.log(e);
  }

  console.log(sock)
  return sock;
}

(function() {
  // The chart labels are the data keys (hours) and the chart data is the values (number of clicks).
  let data = {};

  // Create the chart
  const ctx = document.getElementById('link-detail-chart');
  const chart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: [],
      datasets: [{
        label: 'URL Activity',
        data: [],
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

  // Create a websocket connection
  // TODO: get the link for the detail page.
  const ws = createWebSocket("ws://localhost:8765/v1/updates");
  ws.onmessage = function(message) {
    const click = JSON.parse(message.data);
    if (data[click.time]) {
      data[click.time] += click.views;
    } else {
      data[click.time] = click.views;
    }

    let labels = [];
    let values = [];
    for (const key in data) {
      labels.push(key);
      values.push(data[key]);
    }

    chart.labels = labels;
    chart.data.datasets[0].data = data;
    chart.update();
  };

})();

