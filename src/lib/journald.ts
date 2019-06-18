import { ChildProcess, spawn } from 'child_process';
import { EventEmitter } from 'events';
import StrictEventEmitter from 'strict-event-emitter-types';

import log from './supervisor-console';

export interface Events {
	logLine: Dictionary<unknown>;
	stderrLine: string;
	disconnect: { code: number; signal: string };
	error: Error;
}

export type JournaldEventEmitter = StrictEventEmitter<EventEmitter, Events>;

export class Journald extends (EventEmitter as new () => JournaldEventEmitter) {
	private partialLog: string = '';

	public constructor(private process: ChildProcess) {
		super();
	}

	public end() {
		this.process.kill();
	}

	public addData(data: string) {
		// split on newlines, storing any data which is not
		// followed by a newline
		data = this.partialLog.concat(data);
		this.partialLog = '';
		let index = data.indexOf('\n');
		while (index !== -1) {
			const message = data.slice(0, index);
			this.emitMessage(message);

			data = data.slice(index + 1);
			index = data.indexOf('\n');
		}

		this.partialLog = data;
	}

	private emitMessage(message: string) {
		try {
			const jsonMsg = JSON.parse(message);
			this.emit('logLine', jsonMsg);
		} catch (e) {
			this.emit('error', e);
		}
	}
}

export function getJournald(opts: {
	all: boolean;
	follow: boolean;
	count?: number;
	unit?: string;
}): Journald {
	const args = [
		// The directory we want to run the chroot from
		'/mnt/root',
		'journalctl',
		'-o',
		'json',
	];
	if (opts.all) {
		args.push('-a');
	}
	if (opts.follow) {
		args.push('--follow');
	}
	if (opts.unit != null) {
		args.push('-u');
		args.push(opts.unit);
	}
	if (opts.count != null) {
		args.push('-n');
		args.push(opts.count.toString());
	}

	log.debug('Spawning journald with: chroot ', args.join(' '));

	const journald = spawn('chroot', args, {
		stdio: 'pipe',
	});

	const events = new Journald(journald);

	journald.stdout!.on('data', d => {
		// Parse the message
		events.addData(d.toString());
	});
	journald.stderr.on('data', d => {
		events.emit('stderrLine', d.toString());
	});
	journald.on('close', (code, signal) => {
		events.emit('disconnect', { code, signal });
	});

	return events;
}
