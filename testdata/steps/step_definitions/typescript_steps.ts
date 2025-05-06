import { Given, When, Then, And } from '@cucumber/cucumber';
import { assert } from 'chai';

// Decorator patterns
@Given('I have a TypeScript user with name {string}')
async function createUser(name: string) {
    this.user = { name };
}

@When("I update the TypeScript user's email to {string}")
async function updateEmail(email: string) {
    this.user.email = email;
}

@Then(`the TypeScript user's email should be {string}`)
async function verifyEmail(email: string) {
    assert.equal(this.user.email, email);
}

// Regex patterns with TypeScript types
@Given(/^I have (\d+) TypeScript items$/)
async function createItems(count: number) {
    this.items = Array(count).fill({});
}

@When(/^I add (\d+) more TypeScript items$/)
async function addItems(count: number) {
    this.items.push(...Array(count).fill({}));
}

@Then(/^I should have (\d+) TypeScript items total$/)
async function verifyItemCount(count: number) {
    assert.equal(this.items.length, count);
}

// And variations with TypeScript
@And('the TypeScript user should be active')
async function verifyUserActive() {
    assert.equal(this.user.active, true);
}

@And(/^the TypeScript user should have (\d+) roles$/)
async function verifyUserRoles(count: number) {
    assert.equal(this.user.roles.length, count);
} 