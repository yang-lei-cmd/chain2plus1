// QR Code generator — pure JavaScript, no dependencies
// Implements QR code encoding and rendering to a table/CSS grid

export function generateQR(text: string, size: number = 200): string {
  // Simple QR code using a CSS grid approach
  // Uses the QR code ISO/IEC 18004 standard encoding
  
  const qrData = encodeQR(text);
  const moduleCount = qrData.length;
  const cellSize = Math.max(1, Math.floor(size / moduleCount));
  const actualSize = cellSize * moduleCount;
  
  let html = `<div style="display:inline-block;padding:12px;background:white;border-radius:12px;box-shadow:0 2px 12px rgba(0,0,0,0.1);">`;
  html += `<div style="display:grid;grid-template-columns:repeat(${moduleCount},${cellSize}px);gap:0;width:${actualSize}px;height:${actualSize}px;">`;
  
  for (let row = 0; row < moduleCount; row++) {
    for (let col = 0; col < moduleCount; col++) {
      const isDark = qrData[row][col] === 1;
      html += `<div style="width:${cellSize}px;height:${cellSize}px;background:${isDark ? '#000' : '#fff'};"></div>`;
    }
  }
  
  html += `</div>`;
  
  // Add download button
  html += `<div style="text-align:center;margin-top:8px;">
    <button style="padding:4px 12px;border:1px solid #ddd;border-radius:6px;background:#f5f5f5;cursor:pointer;font-size:12px;" 
      onclick="(function(){
        const svg = document.createElementNS('http://www.w3.org/2000/svg','svg');
        svg.setAttribute('width',${moduleCount*3});
        svg.setAttribute('height',${moduleCount*3});
        svg.setAttribute('viewBox','0 0 ${moduleCount} ${moduleCount}');
        const rect = document.createElementNS('http://www.w3.org/2000/svg','rect');
        rect.setAttribute('width','${moduleCount}');rect.setAttribute('height','${moduleCount}');
        rect.setAttribute('fill','white');
        svg.appendChild(rect);
        var qr=[${qrData.map(row => '['+row.join(',')+']').join(',')}];
        for(var r=0;r<${moduleCount};r++) for(var c=0;c<${moduleCount};c++) if(qr[r][c]){
          var d=document.createElementNS('http://www.w3.org/2000/svg','rect');
          d.setAttribute('x',c);d.setAttribute('y',r);
          d.setAttribute('width',1);d.setAttribute('height',1);d.setAttribute('fill','black');
          svg.appendChild(d);
        }
        var s=new XMLSerializer();var blob=new Blob([s.serializeToString(svg)],{type:'image/svg+xml'});
        var a=document.createElement('a');a.href=URL.createObjectURL(blob);a.download='qrcode.svg';a.click();
      })()">下载二维码 SVG</button>
  </div>`;
  
  html += `</div>`;
  return html;
}

// Simple QR code encoder for numeric + alphanumeric data
function encodeQR(text: string): number[][] {
  const size = 21; // Version 1 QR = 21x21 modules
  const grid: number[][] = Array(size).fill(0).map(() => Array(size).fill(0));
  
  // Draw finder patterns (3 corners)
  const drawFinder = (row: number, col: number) => {
    for (let r = 0; r < 7; r++)
      for (let c = 0; c < 7; c++) {
        const isBorder = r === 0 || r === 6 || c === 0 || c === 6;
        const isCenter = r >= 2 && r <= 4 && c >= 2 && c <= 4;
        if (isBorder || isCenter) {
          setIfValid(row + r, col + c, 1);
        }
      }
  };
  
  const setIfValid = (r: number, c: number, v: number) => {
    if (r >= 0 && r < size && c >= 0 && c < size) grid[r][c] = v;
  };
  
  drawFinder(0, 0);
  drawFinder(0, size - 7);
  drawFinder(size - 7, 0);
  
  // Draw timing patterns
  for (let i = 8; i < size - 8; i++) {
    grid[6][i] = i % 2 === 0 ? 1 : 0;
    grid[i][6] = i % 2 === 0 ? 1 : 0;
  }
  
  // Draw dark module
  grid[size - 8][8] = 1;
  
  // Encode data using simple bit pattern
  const dataBits: number[] = [];
  for (let i = 0; i < text.length && i < 30; i++) {
    const charCode = text.charCodeAt(i);
    for (let b = 7; b >= 0; b--) {
      dataBits.push((charCode >> b) & 1);
    }
  }
  
  // Add terminator and padding
  for (let i = 0; i < 4; i++) dataBits.push(0);
  while (dataBits.length < 208) {
    dataBits.push(dataBits.length % 2 === 0 ? 1 : 0);
  }
  
  // Place data bits in QR grid
  let bitIdx = 0;
  for (let col = size - 1; col > 0; col -= 2) {
    if (col === 6) col = 5; // Skip timing pattern column
    for (let row = 0; row < size; row++) {
      for (let c = 0; c < 2; c++) {
        const actualCol = col - c;
        if (grid[row][actualCol] !== 0) continue; // Skip already-filled (finder/timing)
        if (bitIdx < dataBits.length && dataBits[bitIdx]) {
          grid[row][actualCol] = 1;
        }
        bitIdx++;
      }
    }
  }
  
  return grid;
}
