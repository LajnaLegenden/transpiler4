{ Given, When, Then, And } = require '@cucumber/cucumber'
{ assert } = require 'chai'

# String patterns with different quotes
Given 'I have a CoffeeScript user with name {string}', (name) ->
  @user = { name }

When "I update the CoffeeScript user's email to {string}", (email) ->
  @user.email = email

Then `the CoffeeScript user's email should be {string}`, (email) ->
  assert.equal @user.email, email

# Regex patterns
Given /^I have (\d+) CoffeeScript items$/, (count) ->
  @items = Array(parseInt(count)).fill {}

When /^I add (\d+) more CoffeeScript items$/, (count) ->
  @items.push ...Array(parseInt(count)).fill {}

Then /^I should have (\d+) CoffeeScript items total$/, (count) ->
  assert.equal @items.length, parseInt(count)

# And variations
And 'the CoffeeScript user should be active', ->
  assert.equal @user.active, true

And /^the CoffeeScript user should have (\d+) roles$/, (count) ->
  assert.equal @user.roles.length, parseInt(count) 