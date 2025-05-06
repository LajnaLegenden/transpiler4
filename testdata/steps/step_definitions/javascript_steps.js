const { Given, When, Then, And } = require('@cucumber/cucumber');

// String patterns with different quotes
Given('I have a user with name {string}', function(name) {
    this.user = { name };
});

When("I update the user's email to {string}", function(email) {
    this.user.email = email;
});

Then(`the user's email should be {string}`, function(email) {
    assert.equal(this.user.email, email);
});

// Regex patterns
Given(/^I have (\d+) items$/, function(count) {
    this.items = Array(parseInt(count)).fill({});
});

When(/^I add (\d+) more items$/, function(count) {
    this.items.push(...Array(parseInt(count)).fill({}));
});

Then(/^I should have (\d+) items total$/, function(count) {
    assert.equal(this.items.length, parseInt(count));
});

// And variations
And('the user should be active', function() {
    assert.equal(this.user.active, true);
});

And(/^the user should have (\d+) roles$/, function(count) {
    assert.equal(this.user.roles.length, parseInt(count));
}); 