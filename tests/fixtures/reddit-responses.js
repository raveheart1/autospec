/**
 * @fileoverview Mock Reddit API responses for testing.
 * @author Nicholas C. Zakas
 */

//-----------------------------------------------------------------------------
// Exports
//-----------------------------------------------------------------------------

/**
 * Successful text post submission response
 */
export const SUBMIT_TEXT_SUCCESS = {
	json: {
		errors: [],
		data: {
			url: "https://www.reddit.com/r/test/comments/abc123/hello_reddit/",
			id: "t3_abc123",
			name: "t3_abc123",
		},
	},
};

/**
 * Successful image post submission response
 */
export const SUBMIT_IMAGE_SUCCESS = {
	json: {
		errors: [],
		data: {
			url: "https://www.reddit.com/r/test/comments/xyz789/check_this_out/",
			id: "t3_xyz789",
			name: "t3_xyz789",
		},
	},
};

/**
 * Media asset upload request response (returns S3 upload URL)
 */
export const MEDIA_ASSET_RESPONSE = {
	args: {
		action: "https://reddit-uploaded-media.s3.amazonaws.com/test-bucket",
		fields: [
			{ name: "acl", value: "private" },
			{ name: "key", value: "test-key-12345" },
			{ name: "X-Amz-Credential", value: "test-credential" },
			{ name: "X-Amz-Algorithm", value: "AWS4-HMAC-SHA256" },
			{ name: "X-Amz-Date", value: "20251021T000000Z" },
			{ name: "X-Amz-Security-Token", value: "test-token" },
			{ name: "policy", value: "test-policy" },
			{ name: "X-Amz-Signature", value: "test-signature" },
		],
	},
	asset: {
		asset_id: "test-asset-id-12345",
		processing_state: "pending",
		payload: {
			filepath: "image.png",
		},
		websocket_url: "wss://test.websocket.url",
	},
};

/**
 * S3 upload success response (XML format)
 */
export const S3_UPLOAD_SUCCESS = `<?xml version="1.0" encoding="UTF-8"?>
<PostResponse>
  <Location>https://reddit-uploaded-media.s3.amazonaws.com/test-bucket/test-key-12345</Location>
  <Bucket>reddit-uploaded-media</Bucket>
  <Key>test-key-12345</Key>
  <ETag>"abc123def456"</ETag>
</PostResponse>`;

/**
 * Error: Invalid or expired access token
 */
export const ERROR_USER_REQUIRED = {
	json: {
		errors: [["USER_REQUIRED", "Please log in to do that.", null]],
		data: {},
	},
};

/**
 * Error: Subreddit doesn't exist
 */
export const ERROR_SUBREDDIT_NOEXIST = {
	json: {
		errors: [
			[
				"SUBREDDIT_NOEXIST",
				"Hmm, that community doesn't exist. Try checking the spelling.",
				"sr",
			],
		],
		data: {},
	},
};

/**
 * Error: User not allowed to post in subreddit
 */
export const ERROR_SUBREDDIT_NOTALLOWED = {
	json: {
		errors: [
			["SUBREDDIT_NOTALLOWED", "You aren't allowed to post there.", "sr"],
		],
		data: {},
	},
};

/**
 * Error: Rate limit exceeded
 */
export const ERROR_RATELIMIT = {
	json: {
		errors: [
			[
				"RATELIMIT",
				"You are doing that too much. Try again in 5 minutes.",
				null,
			],
		],
		data: {},
	},
};

/**
 * Error: Title too long
 */
export const ERROR_TOO_LONG = {
	json: {
		errors: [
			["TOO_LONG", "This is too long (max: 300 characters)", "title"],
		],
		data: {},
	},
};

/**
 * Error: Missing title
 */
export const ERROR_NO_TEXT = {
	json: {
		errors: [["NO_TEXT", "We need something here", "title"]],
		data: {},
	},
};
