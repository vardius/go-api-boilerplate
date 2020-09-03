import React, {Component, ReactNode} from "react";
import {defineMessages, injectIntl, IntlShape} from "react-intl";
import {Alert, AlertDescription, AlertIcon, AlertTitle, CloseButton,} from "@chakra-ui/core";

const messages = defineMessages({
  title: {
    id: "app.error.title",
    defaultMessage: "Something went wrong.",
  },
  description: {
    id: "app.error.description",
    defaultMessage: "Please refresh page or try again",
  },
});

export interface Props {
  intl: IntlShape;
  children: ReactNode;
}

type State = {
  readonly error: Error | null;
  readonly errorInfo?: Object;
};

export class ErrorBoundary extends Component<Props, State> {
  readonly state: State = {
    error: null,
  };

  static getDerivedStateFromError(error: Error) {
    return {error};
  }

  componentDidCatch(error: Error, errorInfo: Object): void {
    this.setState({error, errorInfo});
  }

  render() {
    if (this.state.error) {
      return (
        <Alert status="error">
          <AlertIcon/>
          <AlertTitle mr={2}>
            {this.props.intl.formatMessage(messages.title)}
          </AlertTitle>
          <AlertDescription>
            {this.props.intl.formatMessage(messages.description)}
          </AlertDescription>
          <CloseButton position="absolute" right="8px" top="8px"/>
        </Alert>
      );
    }

    return this.props.children;
  }
}

export default injectIntl(ErrorBoundary);
