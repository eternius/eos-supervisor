prepare = require './lib/prepare'
{ expect } = require './lib/chai-config'
constants = require '../src/lib/constants'

describe 'constants', ->
	before ->
		prepare()
	it 'has the correct configJsonPathOnHost', ->
		expect(constants.configJsonPathOnHost).to.equal('/config.json')
	it 'has the correct rootMountPoint', ->
		expect(constants.rootMountPoint).to.equal('./test/data')
