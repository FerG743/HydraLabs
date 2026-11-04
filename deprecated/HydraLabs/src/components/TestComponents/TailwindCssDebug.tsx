<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Tailwind Debug Tool</title>
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        /* Your exact CSS variables */
        @layer base {
          :root {
            --background: 300 15% 6%; /* Deep Purple Black #0F0A0F */
            --foreground: 60 9% 98%; /* Warm White #FBFBF6 */
            --card: 315 20% 9%; /* Dark Charcoal Purple #1A0F1A */
            --card-foreground: 60 9% 98%;
            --primary: 75 100% 50%; /* #80FF00 - Bright Electric Lime */
            --primary-foreground: 300 15% 6%;
            --secondary: 320 60% 15%; /* #3D0B26 */
            --secondary-foreground: 60 9% 98%;
            --muted: 315 20% 12%; /* #241A24 */
            --muted-foreground: 300 15% 65%;
            --accent: 15 100% 60%; /* #FF4D00 - Vibrant Orange-Red */
            --accent-foreground: 60 9% 98%;
            --destructive: 355 85% 60%; /* #E6194D - Hot Pink Red */
            --destructive-foreground: 60 9% 98%;
            --border: 315 20% 18%; /* #3A2B3A */
            --input: 315 20% 18%;
            --ring: 75 100% 50%; /* Electric Lime ring */
          }
        }
        
        body {
            background-color: hsl(var(--background));
            color: hsl(var(--foreground));
            margin: 0;
            padding: 20px;
            font-family: system-ui, -apple-system, sans-serif;
        }
        
        .debug-box {
            padding: 20px;
            margin: 10px 0;
            border: 2px solid hsl(var(--border));
            border-radius: 8px;
        }
    </style>
</head>
<body>
    <div class="debug-box">
        <h1 style="color: hsl(var(--primary));">Tailwind CSS Debug Test</h1>
        <p>If you can see this text clearly on a dark background, your CSS variables are working!</p>
        
        <div class="debug-box" style="background-color: hsl(var(--card));">
            <h2>Color Tests:</h2>
            <p style="color: hsl(var(--primary));">Primary Color (Electric Lime): Should be bright green</p>
            <p style="color: hsl(var(--accent));">Accent Color (Orange-Red): Should be vibrant orange</p>
            <p style="color: hsl(var(--destructive));">Destructive Color (Hot Pink): Should be bright pink</p>
        </div>
        
        <div class="debug-box">
            <h2>Tailwind Classes Test:</h2>
            <div class="bg-primary text-primary-foreground p-4 rounded mb-2">
                Primary Background with Tailwind
            </div>
            <div class="bg-accent text-accent-foreground p-4 rounded mb-2">
                Accent Background with Tailwind
            </div>
            <div class="bg-card text-card-foreground p-4 rounded mb-2">
                Card Background with Tailwind
            </div>
        </div>
        
        <div class="debug-box">
            <h2>Browser Info:</h2>
            <p id="browser-info"></p>
            <p id="css-support"></p>
        </div>
    </div>

    <script>
        document.getElementById('browser-info').innerHTML = `Browser: ${navigator.userAgent}`;
        
        // Test CSS custom property support
        const testEl = document.createElement('div');
        testEl.style.setProperty('--test-var', 'red');
        testEl.style.color = 'var(--test-var)';
        document.body.appendChild(testEl);
        const computedColor = getComputedStyle(testEl).color;
        testEl.remove();
        
        document.getElementById('css-support').innerHTML = `CSS Custom Properties Support: ${computedColor.includes('rgb') ? 'YES' : 'NO'}`;
        
        // Log computed styles for debugging
        console.log('Background color:', getComputedStyle(document.body).backgroundColor);
        console.log('CSS Variables:', {
            background: getComputedStyle(document.documentElement).getPropertyValue('--background'),
            foreground: getComputedStyle(document.documentElement).getPropertyValue('--foreground'),
            primary: getComputedStyle(document.documentElement).getPropertyValue('--primary')
        });
    </script>
</body>
</html>