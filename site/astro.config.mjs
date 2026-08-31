// @ts-check
import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLlmsTxt from 'starlight-llms-txt';

// Cloudflare Pages at a custom domain — base is '/', NOT a project path.
// (toolnexus uses base: '/toolnexus' for GitHub Pages; copying that here would
// break every internal link. Spike S9 asserts links stay root-relative.)
export default defineConfig({
	site: 'https://winctl.deemwar.com',
	integrations: [
		starlight({
			title: 'windowctl',
			description:
				'Move, resize, focus and tile windows across every monitor — one static binary on macOS, Windows and Linux. Plus a device layer that types, clicks, screenshots, captures the screen, and ships its own virtual audio device.',
			tableOfContents: false,
			plugins: [
				starlightLlmsTxt({
					projectName: 'windowctl',
					description:
						'A cross-platform CLI and Go library for window and monitor management, plus a loopback device-streaming layer for auto-testing voice/video apps.',
					details:
						'windowctl lists and moves windows across monitors by zone, split, or coordinates; reports which monitor is active and which window is focused; applies whole layouts in one batch call; and exposes a device layer (keyboard/mouse injection, screenshots, ScreenCaptureKit screen capture, audio device control, speaker loopback, and its own CoreAudio virtual device) over a loopback WebSocket. One static binary per platform, distributed via npm.',
				}),
			],
			customCss: [
				'@fontsource-variable/inter',
				'@fontsource/jetbrains-mono/400.css',
				'@fontsource/jetbrains-mono/600.css',
				'./src/styles/windowctl.css',
			],
			social: [
				{ icon: 'github', label: 'GitHub', href: 'https://github.com/muthuishere/windowctl' },
			],
			sidebar: [
				{
					label: 'Start here',
					items: [
						{ label: 'Quickstart', slug: 'quickstart' },
						{ label: 'Zones & splits', slug: 'zones' },
						{ label: 'Monitors', slug: 'monitors' },
					],
				},
				{
					label: 'Window control',
					items: [
						{ label: 'Listing windows', slug: 'windows' },
						{ label: 'Move, resize, focus', slug: 'move' },
						{ label: 'Layouts in one call', slug: 'batch' },
					],
				},
				{
					label: 'Device layer',
					items: [
						{ label: 'What it is', slug: 'devices' },
						{ label: 'Keyboard & mouse', slug: 'input' },
						{ label: 'Screen & audio', slug: 'capture' },
					],
				},
				{
					label: 'Reference',
					items: [
						{ label: 'Agent skill', slug: 'agent-skill' },
						{ label: 'Go library', slug: 'library' },
						{ label: 'Permissions', slug: 'permissions' },
						{ label: 'How it is built', slug: 'internals' },
					],
				},
			],
		}),
	],
});
