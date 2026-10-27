import { describe, expect, it } from 'vitest';
import { driveRoot, folderName, parentOf, placeOf } from './path';
import { popView, pushView } from './nav';

describe('path helpers', () => {
	it('folderName', () => {
		expect(folderName('D:\\work\\app\\')).toBe('app');
		expect(folderName('D:\\work\\app')).toBe('app');
		expect(folderName('C:\\')).toBe('C:');
	});

	it('parentOf', () => {
		expect(parentOf('D:\\a\\b.txt')).toBe('D:\\a');
		expect(parentOf('D:/a/b')).toBe('D:/a');
	});

	it('driveRoot adds the separator a bare drive needs', () => {
		expect(driveRoot('C:')).toBe('C:\\');
		expect(driveRoot('C:\\')).toBe('C:\\');
		expect(driveRoot('D:\\data')).toBe('D:\\data');
	});

	it('placeOf shows where a result is inside the searched folder', () => {
		expect(placeOf('D:\\work\\a\\b\\file.txt', 'D:\\work')).toBe('a\\b');
		expect(placeOf('D:\\WORK\\x.txt', 'd:\\work\\')).toBe('work');
		expect(placeOf('D:\\work\\x.txt', 'D:\\work', (r) => `${r} (top)`)).toBe('D:\\work (top)');
		expect(placeOf('E:\\other\\x.txt', 'D:\\work')).toBe('E:\\other');
	});
});

describe('navigation stack', () => {
	it('goes forward and back', () => {
		let s: string[] = pushView(['home'], 'tools');
		s = pushView(s, 'slow');
		expect(s).toEqual(['home', 'tools', 'slow']);
		expect(popView(s)).toEqual(['home', 'tools']);
	});

	it('never pops past home', () => {
		expect(popView(['home'])).toEqual(['home']);
	});

	it('returns to a screen already open instead of piling up loops', () => {
		const s = pushView(['home', 'cpu', 'slow'], 'cpu');
		expect(s).toEqual(['home', 'cpu']);
	});

	it('starts over at home', () => {
		expect(pushView(['home', 'tools', 'dupes'], 'home')).toEqual(['home']);
	});
});
