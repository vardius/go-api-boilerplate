import React from 'react';
import {Box} from 'src/components/bootstrap';
import CardBody from './components/CardBody';
import CardFooter from './components/CardFooter';
import CardHeader from './components/CardHeader';
import styles from './Card.module.scss';

export interface Props {
  className?: string;
}

function Card({className, ...otherProps}: Props) {
  const scssClasses = [styles.card];

  if (className) {
    scssClasses.push(className);
  }

  return <Box className={scssClasses.join(' ')} {...otherProps} />;
}

Card.Body = CardBody;
Card.Footer = CardFooter;
Card.Header = CardHeader;

export default Card;
