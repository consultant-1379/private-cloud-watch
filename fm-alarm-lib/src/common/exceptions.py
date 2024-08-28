"""Define exception handling."""


class ValidationException(Exception):
    """Define custom exception for arg validation."""

    def __init__(self, message):
        """
        Init for ValidationException object.

        :param message: error message.
        """
        super(ValidationException, self).__init__(message)
        self.message = message

    def __str__(self):
        """Define the string representation of object."""
        return "ValidationException: {}".format(self.message)


class ConfigValidationException(Exception):
    """Define custom exception for arg validation."""

    def __init__(self, message):
        """
        Init for ConfigValidationException object.

        :param message: error message.
        """
        super(ConfigValidationException, self).__init__(message)
        self.message = message

    def __str__(self):
        """Define the string representation of object."""
        return "ConfigValidationException: {}".format(self.message)


class ConfigParsingException(Exception):
    """Define custom exception for arg validation."""

    def __init__(self, message):
        """
        Init for ConfigParsingException object.

        :param message: error message.
        """
        super(ConfigParsingException, self).__init__(message)
        self.message = message

    def __str__(self):
        """Define the string representation of object."""
        return "ConfigValidationException: {}".format(self.message)


class EnmAlarmException(Exception):
    """Define custom exception for alarm."""

    def __init__(self, message):
        """
        Init for EnmAlarmException object.

        :param message: error message.
        """
        super(EnmAlarmException, self).__init__(message)
        self.message = message

    def __str__(self):
        """Define the string representation of object."""
        return "EnmAlarmException: {}".format(self.message)


class ConsulUtilsException(Exception):
    """Define custom exception for alarm."""

    def __init__(self, message):
        """
        Init for ConsulException object.

        :param message: error message.
        """
        super(ConsulUtilsException, self).__init__(message)
        self.message = message

    def __str__(self):
        """Define the string representation of object."""
        return "ConsulException: {}".format(self.message)
