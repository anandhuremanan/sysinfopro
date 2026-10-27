import { describe, expect, it } from 'vitest';
import { makeRedactor } from './redact';

const redact = makeRedactor(['dev', 'LAPTOP-7Q2', 'SN12345']);

describe('makeRedactor', () => {
	it('hides the user folder in paths, whatever the name', () => {
		expect(redact('C:\\Users\\ANAND~1\\AppData')).toBe('C:\\Users\\(hidden)\\AppData');
		expect(redact('d:/users/someone/x')).toBe('d:/users/(hidden)/x');
		// JSON doubles each backslash
		expect(redact('"C:\\\\Users\\\\al\\\\Desktop"')).toBe('"C:\\\\Users\\\\(hidden)\\\\Desktop"');
	});

	it('matches names only as whole words', () => {
		expect(redact('Developer tools on dev')).toBe('Developer tools on (hidden)');
		expect(redact('PC LAPTOP-7Q2, serial SN12345')).toBe('PC (hidden), serial (hidden)');
	});

	it('hides even one-letter user names when they stand alone', () => {
		expect(makeRedactor(['a'])('user a signed in, again')).toBe('user (hidden) signed in, again');
	});

	it('hides emails', () => {
		expect(redact('signed in as jane.doe@example.com')).toBe('signed in as (email hidden)');
	});

	it('hides IPv4 addresses but keeps version numbers', () => {
		expect(redact('router 192.168.1.1 answered')).toBe('router (hidden) answered');
		expect(redact('address 10.0.0.5.')).toBe('address (hidden).');
		expect(redact('driver 31.0.101.5081')).toBe('driver 31.0.101.5081');
		expect(redact('Windows 10.0.22631')).toBe('Windows 10.0.22631');
		expect(redact('v1.2.3.4')).toBe('v1.2.3.4');
	});

	it('hides IPv6 addresses, but not times', () => {
		expect(redact('dns fe80::1c2a:3bff:fe4d:5e6f%12')).toBe('dns (hidden)');
		expect(redact('2001:0db8:85a3:0000:0000:8a2e:0370:7334 ok')).toBe('(hidden) ok');
		expect(redact('at 12:30:45 today')).toBe('at 12:30:45 today');
	});

	it('hides MAC addresses with or without separators', () => {
		expect(redact('mac 00-15-5D-01-23-45')).toBe('mac (hidden)');
		expect(redact('mac 00155D0123AB')).toBe('mac (hidden)');
	});

	it('leaves text alone when there is nothing to hide', () => {
		expect(makeRedactor([])('16 GB · 3200 MT/s')).toBe('16 GB · 3200 MT/s');
	});
});
