package main

const WebPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<title>LAN-Share</title>
<style>
	html, body {
		margin: 0;
		height: 100vh;
		width: 100vw;
	}
	body {
		display: grid;
		grid-template: 1fr 64px / 1fr;
	}
	#history {
		display: flex;
		flex-direction: column;
		overflow-y: scroll;
		padding: 8px 8px 0;
	}
	#form {
		position: relative;
	}
	#sender {
		width: 100%;
		height: 100%;
		display: grid;
		grid-template: 1fr / 1fr 128px;
	}
	#buttons {
		display: grid;
		grid-template: 'a b' 1fr 'c c' 1fr / 1fr 1fr;
	}
	textarea, button {
		width: 100%;
		height: 100%;
		padding: 0;
		margin: 0;
		box-sizing: border-box;
		appearance: none;
		border: 1px solid #ccc;
		background-color: transparent;
		outline: none;
	}
	textarea {
		padding: 8px;
		resize: none;
	}
	button:nth-child(1) {
		grid-area: a;
	}
	button:nth-child(2) {
		grid-area: b;
	}
	button:nth-child(3) {
		grid-area: c;
	}
	button:hover {
		background-color: #eee;
		cursor: pointer;
	}
	#file-selector {
		display: none;
	}
	#tip {
		position: absolute;
		bottom: 8px;
		right: 136px;
		font-size: 12px;
		color: #888;
	}
	#connecting {
		position: fixed;
		left: 0;
		bottom: 0;
		width: 100vw;
		height: 64px;
		background-color: #aaa5;
		font-size: 3em;
		display: flex;
		align-items: center;
		justify-content: center;
	}
</style>
</head>
<body>
<div id="history"></div>
<template id="message">
<style>
	:host {
		display: block;
		margin-bottom: 8px;
		padding: 8px;
		border-radius: 8px;
		background-color: #ccc;
	}
	::slotted([slot=name]) {
		font-size: 1.3em;
		font-weight: bold;
		margin-right: 8px;
		line-height: 30px;
	}
	::slotted([slot=time]) {
		font-style: italic;
		line-height: 30px;
	}
	header {
		display: flex;
		user-select: none;
	}
	img {
		max-width: 100%;
	}
	table {
		width: 100%;
		text-align: center;
	}
</style>
<header>
	<slot name="name"></slot>
	<slot name="time"></slot>
</header>
<main></main>
</template>
<form id="form">
	<div id="sender">
		<textarea name="text" placeholder="Input your text message here"></textarea>
		<div id="buttons">
			<button id="image" type="button">Image</button>
			<button id="file" type="button">File</button>
			<button type="submit">Send</button>
		</div>
	</div>
	<input id="file-selector" type="file">
	<div id="tip">Press Shift+Enter to send</div>
</form>
<div id="connecting"><span>Connecting......</span></div>
<script>
(() => {
const history = document.getElementById('history');
const form = document.getElementById('form');
const textarea = document.querySelector('textarea[name=text]');
const image = document.getElementById('image');
const file = document.getElementById('file');
const connecting = document.getElementById('connecting');
const fileSelector = document.getElementById('file-selector');
const fileHolder = {};
const MsgType = {
	Text: 0,
	Image: 1,
	File: 2,
	ClearFile: 3,
	RequestFile: 4,
};
const query = new URLSearchParams(location.search);
const wsURL = new URL(` + "`" + `/ws${query.get('name') ? ` + "`" + `?name=${query.get('name')}` + "`" + ` : ''}` + "`" + `, location.href);
wsURL.protocol = wsURL.protocol === 'https' ? 'wss' : 'ws';
let ws;
const connect = () => {
	ws = new WebSocket(wsURL.toString());
	ws.addEventListener('open', () => {
		history.innerHTML = '';
		connecting.style.display = 'none';
		textarea.focus();
	});
	ws.addEventListener('close', () => {
		ws = undefined;
		connecting.style.display = '';
		textarea.blur();
		setTimeout(connect, 1e3);
	});
	ws.addEventListener('message', async ({ data }) => {
		const arrayBuffer = await data.arrayBuffer();
		const view = new DataView(arrayBuffer);
		let offset = 0;
		const type = view.getUint8(offset++);
		if (type === MsgType.RequestFile) {
			let id = 0;
			for (let i = 24; i >= 0; i -= 8) {
				id += view.getUint8(offset++) * (2 ** i);
			}
			const file = fileHolder[id]?.file;
			if (file) {
				let range;
				let contentRange;
				let fileRange;
				if (offset < view.byteLength) {
					range = decoder.decode(arrayBuffer.slice(offset));
					const match = range.split(',')[0].match(/^(?<unit>[^=]+)=(?<start>\d*)-(?<end>\d*)/);
					if (match) {
						if (match.groups.start) {
							fileRange = [Number(match.groups.start), match.groups.end ? Number(match.groups.end) + 1 : file.size];
							contentRange = ` + "`" + `${match.groups.unit} ${fileRange[0]}-${fileRange[1] - 1}/${file.size}` + "`" + `;
						} else if (match.groups.end) {
							fileRange = [file.size - Number(match.groups.end), file.size];
							contentRange = ` + "`" + `${match.groups.unit} ${fileRange[0]}-${fileRange[1] - 1}/${file.size}` + "`" + `;
						}
					}
				}
				const query = new URLSearchParams({
					name: file.name,
					size: fileRange ? fileRange[1] - fileRange[0] : file.size,
					type: file.type,
				});
				if (range) {
					query.set('range', range);
				}
				fetch(` + "`" + `/upload/${id}?${query.toString()}` + "`" + `, {
					method: 'POST',
					headers: {
						'Content-Type': file.type,
						...(contentRange ? { 'Content-Range': contentRange } : {}),
					},
					body: fileRange ? file.slice(fileRange[0], fileRange[1]) : file,
				}).catch(console.error);
			}
			return;
		}
		if (type === MsgType.ClearFile) {
			while (offset < view.byteLength) {
				let id = 0;
				for (let i = 24; i >= 0; i -= 8) {
					id += view.getUint8(offset++) * (2 ** i);
				}
				const ele = history.querySelector(` + "`" + `[data-file="${id}"]` + "`" + `);
				ele?.remove();
			}
			return;
		}
		const nameLen = view.getUint8(offset++);
		const name = document.createElement('div');
		name.slot = 'name';
		name.innerHTML = decoder.decode(arrayBuffer.slice(offset, offset + nameLen));
		offset += nameLen;
		let scrollTimeout = 0;
		let timestamp = 0;
		for (let i = 56; i >= 0; i -= 8) {
			timestamp += view.getUint8(offset++) * (2 ** i);
		}
		const time = document.createElement('time');
		time.slot = 'time';
		const date = new Date(timestamp);
		time.dateTime = date.toJSON();
		time.innerHTML = date.toLocaleString();
		const msg = document.createElement('lan-share-msg');
		msg.dataset.time = timestamp;
		msg.appendChild(name);
		msg.appendChild(time);
		switch (type) {
		case MsgType.Text:
			msg.setText(decoder.decode(arrayBuffer.slice(offset)));
			break;
		case MsgType.Image:
			const imageTypeLen = view.getUint8(offset++);
			const imageType = decoder.decode(arrayBuffer.slice(offset, offset + imageTypeLen));
			offset += imageTypeLen;
			msg.setImage(imageType, arrayBuffer.slice(offset));
			break;
		case MsgType.File:
			let id = 0;
			for (let i = 24; i >= 0; i -= 8) {
				id += view.getUint8(offset++) * (2 ** i);
			}
			msg.dataset.file = id;
			const info = JSON.parse(decoder.decode(arrayBuffer.slice(offset)));
			msg.setFile(id, info);
			break;
		}
		const children = history.children;
		let target = children[0] || null;
		for (let i = children.length - 1; i >= 0; --i) {
			if (Number(children[i].dataset.time) < timestamp) {
				target = children[i + 1] || null;
				break;
			}
		}
		history.insertBefore(msg, target);
		if (scrollTimeout) {
			clearTimeout(scrollTimeout);
		}
		scrollTimeout = setTimeout(() => {
			msg.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
		}, 0.1e3);
	});
};
connect();
const encoder = new TextEncoder();
const decoder = new TextDecoder();
form.addEventListener('submit', e => {
	e.preventDefault();
	if (!ws) {
		return;
	}
	const text = new FormData(e.target).get('text') || '';
	e.target.reset();
	const u8arr = encoder.encode(text);
	if (u8arr.length) {
		ws.send(new Blob([Uint8Array.from([MsgType.Text]), u8arr]));
	}
});
window.addEventListener('keydown', e => {
	if (!e.shiftKey || e.ctrlKey || e.altKey || e.metaKey) {
		return;
	}
	if (e.key === 'Enter') {
		e.preventDefault();
		form.requestSubmit();
	}
});
let currentSelect;
fileSelector.addEventListener('change', async () => {
	if (!fileSelector.files.length) {
		return;
	}
	switch (currentSelect) {
	case 'image':
		for (const file of fileSelector.files) {
			const u8arr = encoder.encode(file.type);
			ws.send(new Blob([Uint8Array.from([MsgType.Image, u8arr.length]), u8arr, file]));
		}
		break;
	case 'file':
		for (const file of fileSelector.files) {
			const idRes = await fetch('/id');
			const { id } = await idRes.json();
			fileHolder[id] = {
				name: file.name,
				type: file.type,
				size: file.size,
				updated: file.lastModified,
				file,
			};
			const u8ID = new Uint8Array(4);
			let tmpID = id;
			for (let i = 3; i >= 0; --i) {
				u8ID[i] = tmpID & 0xFF;
				tmpID >>= 8;
			}
			const u8arr = encoder.encode(JSON.stringify(fileHolder[id], (k, v) => k === 'file' ? undefined : v));
			ws.send(new Blob([Uint8Array.from([MsgType.File]), u8ID, u8arr]));
		}
		break;
	}
	fileSelector.value = '';
})
image.addEventListener('click', () => {
	currentSelect = 'image';
	fileSelector.click();
});
file.addEventListener('click', () => {
	currentSelect = 'file';
	fileSelector.click();
});
})();
const messageTmpl = document.getElementById('message');
const byteUnit = ['KiB', 'MiB', 'GiB'];
window.customElements.define('lan-share-msg', class extends HTMLElement {
	#main = null;
	#url = null;
	constructor() {
		super();
		this.attachShadow({ mode: 'open' });
		this.shadowRoot.appendChild(messageTmpl.content.cloneNode(true));
		this.#main = this.shadowRoot.querySelector('main');
	}
	disconnectedCallback() {
		this.release();
	}
	release() {
		if (this.#url) {
			URL.revokeObjectURL(this.#url);
			this.#url = null;
		}
	}
	setText(text) {
		this.release();
		this.#main.innerHTML = text;
	}
	setImage(type, buffer) {
		this.release();
		this.#url = URL.createObjectURL(new Blob([buffer], { type }));
		const image = new Image();
		image.src = this.#url;
		this.#main.innerHTML = '';
		this.#main.appendChild(image);
	}
	setFile(id, info) {
		this.release();
		let sizeText = '';
		for (let i = 0, size = info.size / 1024; size >= 1 && i < byteUnit.length; size /= 1024, ++i) {
			sizeText = ` + "`" + `${size.toFixed(2)}${byteUnit[i]}` + "`" + `;
		}
		const date = new Date(info.updated);
		this.#main.innerHTML = ` + "`" + `<table>
<thead>
<tr>
	<th>Filename</th>
	<th>Size</th>
	<th>Type</th>
	<th>Last Modified</th>
	<th></th>
</tr>
</thead>
<tbody>
<tr>
	<td>${info.name}</td>
	<td>${sizeText ? ` + "`" + `${sizeText} (${info.size})` + "`" + ` : info.size}</td>
	<td>${info.type}</td>
	<td><time dateTime="${date.toJSON()}">${date.toLocaleString()}</time></td>
	<td><a href="/download/${id}?open" target="_blank">Open</a> <a href="/download/${id}" download>Download</a></td>
</tr>
</tbody>
		</table>` + "`" + `;
	}
});
</script>
</body>
</html>`
