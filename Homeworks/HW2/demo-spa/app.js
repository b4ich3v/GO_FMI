const root = document.getElementById('root');

const img1 = document.createElement('img');
img1.alt = 'gopher';
img1.title = 'Injected by JS';
img1.src = 'https://go.dev/images/gophers/motorcycle.svg';
root.appendChild(img1);

const img2 = document.createElement('img');
img2.alt = 'go logo';
img2.title = 'Injected by JS';
img2.src = 'https://go.dev/images/go-logo-blue.svg';
root.appendChild(img2);
