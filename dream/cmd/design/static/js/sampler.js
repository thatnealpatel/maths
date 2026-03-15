document.addEventListener("DOMContentLoaded", function () {
  var fonts = {
    ibm: "'IBM Plex Mono', monospace",
    menlo: "'Menlo', monospace",
    pt: "'PT Mono', monospace",
  };

  var buttons = document.querySelectorAll(".FontToggle .Btn");
  buttons.forEach(function (btn) {
    btn.addEventListener("click", function () {
      var key = btn.getAttribute("data-font");
      document.documentElement.style.setProperty("--font-family", fonts[key]);
      buttons.forEach(function (b) { b.classList.remove("active"); });
      btn.classList.add("active");
      drawPlot();
    });
  });

  var freqSlider = document.getElementById("freq-slider");
  var freqReadout = document.getElementById("freq-readout");
  freqSlider.addEventListener("input", function () {
    freqReadout.textContent = parseFloat(freqSlider.value).toFixed(2);
    drawPlot();
  });

  var canvas = document.getElementById("plot-canvas");
  var ctx = canvas.getContext("2d");

  function drawPlot() {
    var dpr = window.devicePixelRatio || 1;
    var rect = canvas.getBoundingClientRect();
    canvas.width = rect.width * dpr;
    canvas.height = rect.height * dpr;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);

    var w = rect.width;
    var h = rect.height;
    var freq = parseFloat(freqSlider.value);

    var style = getComputedStyle(document.documentElement);
    var rule = style.getPropertyValue("--rule").trim();
    var inkFaint = style.getPropertyValue("--ink-faint").trim();
    var fontFamily = style.getPropertyValue("--font-family").trim();

    var xMin = -12, xMax = 12;
    var yMin = -0.5, yMax = 1.2;

    function toCanvasX(x) { return (x - xMin) / (xMax - xMin) * w; }
    function toCanvasY(y) { return h - (y - yMin) / (yMax - yMin) * h; }

    ctx.clearRect(0, 0, w, h);

    ctx.strokeStyle = rule;
    ctx.lineWidth = 0.5;
    for (var gx = Math.ceil(xMin); gx <= Math.floor(xMax); gx++) {
      var cx = toCanvasX(gx);
      ctx.beginPath();
      ctx.moveTo(cx, 0);
      ctx.lineTo(cx, h);
      ctx.stroke();
    }
    for (var gy = -0.4; gy <= 1.2; gy += 0.2) {
      var cy = toCanvasY(gy);
      ctx.beginPath();
      ctx.moveTo(0, cy);
      ctx.lineTo(w, cy);
      ctx.stroke();
    }

    ctx.strokeStyle = inkFaint;
    ctx.lineWidth = 1;
    var zeroY = toCanvasY(0);
    ctx.beginPath();
    ctx.moveTo(0, zeroY);
    ctx.lineTo(w, zeroY);
    ctx.stroke();
    var zeroX = toCanvasX(0);
    ctx.beginPath();
    ctx.moveTo(zeroX, 0);
    ctx.lineTo(zeroX, h);
    ctx.stroke();

    ctx.font = "13px " + fontFamily;
    ctx.fillStyle = inkFaint;
    ctx.textAlign = "center";
    for (var lx = Math.ceil(xMin); lx <= Math.floor(xMax); lx++) {
      if (lx === 0) continue;
      ctx.fillText(lx.toString(), toCanvasX(lx), zeroY + 14);
    }
    ctx.textAlign = "right";
    for (var ly = -0.4; ly <= 1.2; ly += 0.2) {
      var rounded = Math.round(ly * 10) / 10;
      if (rounded === 0) continue;
      ctx.fillText(rounded.toFixed(1), zeroX - 6, toCanvasY(ly) + 3);
    }

    var steps = 800;
    var dx = (xMax - xMin) / steps;

    ctx.strokeStyle = "#C0522A";
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    var started = false;
    for (var i = 0; i <= steps; i++) {
      var x = xMin + i * dx;
      var y = x === 0 ? 1 : Math.sin(x) / x;
      var px = toCanvasX(x);
      var py = toCanvasY(y);
      if (!started) { ctx.moveTo(px, py); started = true; }
      else { ctx.lineTo(px, py); }
    }
    ctx.stroke();

    ctx.strokeStyle = "#2E7D8C";
    ctx.lineWidth = 1.5;
    ctx.setLineDash([5, 4]);
    ctx.beginPath();
    started = false;
    for (var j = 0; j <= steps; j++) {
      var x2 = xMin + j * dx;
      var y2 = Math.cos(2 * Math.PI * freq * x2) * 0.5;
      var px2 = toCanvasX(x2);
      var py2 = toCanvasY(y2);
      if (!started) { ctx.moveTo(px2, py2); started = true; }
      else { ctx.lineTo(px2, py2); }
    }
    ctx.stroke();
    ctx.setLineDash([]);

    var legendX = 14;
    var legendY = 18;
    ctx.font = "13px " + fontFamily;

    ctx.strokeStyle = "#C0522A";
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    ctx.moveTo(legendX, legendY);
    ctx.lineTo(legendX + 20, legendY);
    ctx.stroke();
    ctx.fillStyle = "#C0522A";
    ctx.textAlign = "left";
    ctx.fillText("sin(x)/x", legendX + 26, legendY + 3);

    ctx.strokeStyle = "#2E7D8C";
    ctx.setLineDash([5, 4]);
    ctx.beginPath();
    ctx.moveTo(legendX, legendY + 16);
    ctx.lineTo(legendX + 20, legendY + 16);
    ctx.stroke();
    ctx.setLineDash([]);
    ctx.fillStyle = "#2E7D8C";
    ctx.fillText("cos(2\u03C0\u03BEx)\u00B70.5", legendX + 26, legendY + 19);
  }

  drawPlot();
  window.addEventListener("resize", drawPlot);
});
